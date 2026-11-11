# YouTube Cache Proxy

Service Go en cours de développement pour mettre en cache l'audio de vidéos YouTube.

## État actuel

Le dépôt contient les fondations du backend :

- un serveur HTTP avec arrêt propre sur `SIGINT` et `SIGTERM` ;
- une API REST pour ajouter, filtrer, consulter et supprimer des sources, lancer un lot manuel, relancer une demande échouée, servir l'audio et contrôler le worker ;
- une extraction des métadonnées YouTube sans audio à l'ajout : chaîne, miniature, publication, description, tags, catégories, vues, langue et état du direct ;
- `GET /health`, qui renvoie `{"status":"ok"}` ;
- une configuration par variables d'environnement, avec validation des limites de téléchargement et de reprise ;
- un stockage SQLite avec migrations et opérations de file d'attente ;
- une suppression atomique des sources, refusée tant qu'elles appartiennent à un lot actif ;
- un composant interne d'extraction des métadonnées et de téléchargement audio via `yt-dlp`, avec limites de durée, taille, débit et délai d'exécution ;
- un moteur interne qui réserve un lot manuel, traite les téléchargements en arrière-plan, respecte le quota glissant sur 24 h et attend une pause aléatoire entre tentatives ;
- une classification des erreurs `yt-dlp` pour distinguer les échecs réseau relançables des erreurs définitives.

**L'API de base et un tableau de bord sont disponibles, mais le projet reste en développement.** Le tableau de bord est accessible à `/` et `/dashboard` ; il permet d'ajouter une vidéo, démarrer un lot, relancer ou supprimer une demande, et mettre le worker en pause ou le reprendre. Il affiche les compteurs par état, les réussites sur 24 h et une liste paginée de 20 sources. Chaque source peut afficher ses métadonnées YouTube et son erreur éventuelle. La liste s'adapte aux petits écrans. La connexion échange le jeton administrateur contre un cookie de session signé, HttpOnly, Secure et limité à huit heures, ou 30 jours avec « Rester connecté » ; le jeton n'est pas conservé dans le navigateur. Les requêtes qui modifient l'état exigent aussi un jeton CSRF lié à la session. Son actualisation automatique s'interrompt lorsque l'onglet est masqué. Les routes REST listées ci-dessous sont disponibles, notamment la récupération des fichiers audio. Les erreurs REST utilisent un format JSON stable ; ne l'expose pas sur un réseau non fiable.

Une image Docker multi-architecture `linux/amd64` et `linux/arm64` est disponible via le Dockerfile. Elle inclut Go 1.27.3 au build, Python 3.13.18, yt-dlp 2026.10.26, yt-dlp-ejs 0.8.0, Deno 2.9.8 et FFmpeg 5.1.9. Le service s'exécute sans privilèges root et utilise `/data` pour la base SQLite et les fichiers audio. La configuration Compose fournit un volume persistant et un healthcheck.

L'API exige un jeton Bearer : le rôle `producer` peut ajouter et consulter les sources et récupérer l'audio ; le rôle `admin` peut aussi démarrer des lots, relancer ou supprimer une source et contrôler le worker. `/health` reste public. L'endpoint audio utilise le service de fichiers Go, qui prend en charge les requêtes Range.

La pause laisse la vidéo en cours se terminer, puis conserve le lot actif pour la reprise. Au redémarrage du service, un lot actif est marqué `interrupted` et ses sources inachevées redeviennent `pending` ; aucun téléchargement ne reprend sans nouveau lancement explicite.

`POST /sources` lance une requête yt-dlp limitée aux métadonnées (30 secondes maximum), sans téléchargement audio. Si YouTube ou yt-dlp ne renvoie pas les détails, la source reste créée en `pending` et les champs indisponibles sont omis. Les métadonnées enrichies apparaissent également dans les réponses de liste et de détail.

Configure `PRODUCER_TOKEN`, `ADMIN_TOKEN` et `SESSION_SECRET` avec trois valeurs distinctes d'au moins 32 caractères. Ces trois variables sont obligatoires au démarrage.

## API REST actuelle

| Méthode | Endpoint | Description |
|---|---|---|
| `POST` | `/sources` | Producer/Admin : ajouter une URL YouTube et récupérer ses métadonnées (`{"url":"https://..."}`) |
| `GET` | `/sources?status=pending&video_id=abc123def45&limit=50&offset=0` | Producer/Admin : lister et filtrer les sources par statut et identifiant vidéo |
| `GET` | `/sources/{id}` | Producer/Admin : lire une demande et ses métadonnées |
| `GET` | `/sources/{id}/audio` | Producer/Admin : récupérer l'audio prêt, avec support Range natif |
| `POST` | `/downloads/run` | Admin : démarrer un lot asynchrone (`{"count":3}`) |
| `POST` | `/sources/{id}/retry` | Admin : remettre une demande échouée en attente |
| `DELETE` | `/sources/{id}` | Admin : supprimer une demande et son fichier audio |
| `POST` | `/worker/pause` | Admin : empêcher le démarrage de la prochaine vidéo |
| `POST` | `/worker/resume` | Admin : reprendre un lot actif |
| `GET` | `/worker/status` | Admin : lire les compteurs et l'état du worker |
| `GET` | `/health` | Public : vérifier la réponse HTTP |

Les erreurs REST renvoient `{"code":"bad_request","message":"invalid JSON request"}` avec un code stable adapté au statut HTTP. Les erreurs internes ne divulguent pas leur cause. Les corps JSON sont limités à 1 Mio (au-delà, `413 payload_too_large`). Une plage audio invalide renvoie un DTO JSON `416 range_not_satisfiable` tout en conservant l’en-tête `Content-Range`.

## Lancer le serveur

Prérequis : Go 1.27 ou supérieur.

```sh
go run ./cmd/server
```

Par défaut, le serveur écoute sur `127.0.0.1:8080`. Vérifier son état :

```sh
curl http://127.0.0.1:8080/health
```

Réponse attendue :

```json
{"status":"ok"}
```

Le serveur lit sa configuration depuis l'environnement. Les valeurs par défaut incluent `HTTP_ADDR=127.0.0.1:8080`, `SQLITE_PATH=data/queue.db` et `AUDIO_DIR=data/audio`. Les autres paramètres et leurs contraintes sont définis dans `internal/config`.

Lorsqu'il est lancé directement hors conteneur, le binaire `yt-dlp` doit être installé et disponible dans `PATH` pour traiter les lots démarrés via l'API. L'image Docker fournit yt-dlp, Deno et FFmpeg.

## Construire et lancer l'image Docker

Avec Docker Buildx, construire l'image ARM64 destinée notamment au Raspberry Pi :

```sh
docker buildx build --platform linux/arm64 --load -t youtube-cache-proxy:local .
```

Sur une machine ARM64, `docker build -t youtube-cache-proxy:local .` utilise directement son architecture native. Pour lancer le conteneur, fournir les deux jetons et le secret de session requis :

```sh
docker run --rm -p 127.0.0.1:8081:8080 \
  -e PRODUCER_TOKEN="<jeton-producer-de-32-caracteres-minimum>" \
  -e ADMIN_TOKEN="<jeton-admin-different-de-32-caracteres-minimum>" \
  -e SESSION_SECRET="<secret-session-different-des-jetons>" \
  -v youtube-cache-data:/data \
  youtube-cache-proxy:local
```

L'image écoute sur `0.0.0.0:8080` et conserve SQLite et les fichiers audio dans `/data`. Le fichier Compose ci-dessous utilise les mêmes chemins.

## Lancer avec Docker Compose

Copier `.env.example` vers `.env`, puis renseigner trois secrets distincts d'au moins 32 caractères. Exécuter `openssl rand -hex 32` trois fois et copier un résultat différent dans `PRODUCER_TOKEN`, `ADMIN_TOKEN` et `SESSION_SECRET`. Chaque résultat contient 64 caractères aléatoires ; ne pas utiliser de mot de passe inventé. Le fichier `.env` est ignoré par Git et exclu du contexte de build. Protéger ses permissions avec `chmod 600 .env`.

```sh
cp .env.example .env
# Renseigner PRODUCER_TOKEN, ADMIN_TOKEN et SESSION_SECRET dans .env.
chmod 600 .env
docker compose up -d --build
docker compose ps
curl --fail http://127.0.0.1:8081/health
```

Ouvrir `http://127.0.0.1:8081/` dans un navigateur pour utiliser le tableau de bord. Saisir le jeton `ADMIN_TOKEN` pour ouvrir une session valable 30 jours avec « Rester connecté » coché (huit heures si décoché). Il est échangé contre un cookie signé et protégé ; le navigateur ne conserve pas le jeton. Pour utiliser le cookie `Secure`, servir le tableau de bord en HTTPS.

Le port est publié uniquement sur `127.0.0.1:8081` de l'hôte par défaut, pour laisser le port 8080 disponible aux autres projets. Pour choisir un autre port libre, modifier `HOST_PORT=8081` dans `.env`, puis exécuter `docker compose up -d` et utiliser ce port dans les URL. Le port interne reste `8080`. Le healthcheck interroge `/health` toutes les 30 secondes depuis le conteneur, sans outil supplémentaire. `restart: unless-stopped` relance le service après une sortie du processus ou un redémarrage Docker ; un état `unhealthy` seul ne déclenche pas de redémarrage.

Le volume nommé `cache-data`, préfixé par le nom du projet Compose, conserve SQLite et ses fichiers WAL ainsi que les audios sous `/data`. Il reste présent après `docker compose down` et une recréation du conteneur. Garder le même nom de projet pour retrouver ces données. **`docker compose down -v` supprime le volume et toutes ses données.**

Consulter les logs avec `docker compose logs --tail=100 youtube-cache-proxy` et arrêter le service avec `docker compose down`. Aucun téléchargement ne démarre automatiquement : le lancement d'un lot reste une action admin explicite.

## Vérifications

```sh
go test ./...
go vet ./...
```

## Cache de découverte — Phase 1

Le cache de découverte est indépendant de la file audio : une recherche ne crée
aucun téléchargement. Configurer `YOUTUBE_API_KEY` pour activer les recherches
YouTube Data API v3. Sans clé, les lectures restent disponibles et le lancement
renvoie 503. La clé reste côté serveur.

| Méthode | Endpoint | Accès et contrat |
|---|---|---|
| POST | /discovery/searches | Admin, corps ci-dessous, réponse 202 et Location |
| GET | /discovery/searches/{id} | Producer/Admin, statut persistant et erreur |
| GET | /discovery/videos | Producer/Admin, lecture SQLite paginée |

Exemple de corps :
```json
{"query":"énergie solaire","channelId":"","publishedAfter":"2026-01-01T00:00:00Z","publishedBefore":"2026-11-01T00:00:00Z"}
```

Au moins `query` ou `channelId` est requis. Une recherche effectue au maximum une
page de 50 résultats (`search.list`, coût YouTube de 100 unités), puis un appel
`videos.list` groupé pour obtenir les métadonnées et durées (1 unité).
Une nouvelle demande POST est une nouvelle exécution explicitement facturable.
Les vidéos sont uniques par identifiant YouTube et associées à chaque recherche.
Aucun accès réel à YouTube n'est nécessaire aux tests.

La liste retourne `{"items":[],"limit":50,"offset":0}`. Filtres :
`searchId`, `channelId`, `publishedAfter`, `publishedBefore`,
`limit` (1–100, défaut 50), `offset` (défaut 0), `sort`
(`-publishedAt` par défaut ou `publishedAt`). L'ordre inclut l'identifiant vidéo
pour stabiliser les pages. Les vidéos contiennent titre, description, chaîne,
miniature, publication, durée ISO 8601 et date de mise en cache. Le lien YouTube
se construit avec `https://www.youtube.com/watch?v={videoId}`.
Les associations d'origine sont consultables avec `searchId` et le détail de recherche.

Le worker traite une recherche à la fois, attend une seconde après chaque recherche
et dort sur une notification lorsqu'il est inactif. Les appels HTTP utilisent un
client réutilisé, avec délai réseau de 30 secondes et délai global de 45 secondes
par recherche. Les résultats et le statut completed sont écrits dans une transaction.
Les statuts sont queued, running, completed, failed et interrupted. Après redémarrage,
les recherches queued sont reprises ; les running deviennent interrupted, sans
répéter un appel potentiellement consommé. L'arrêt annule l'appel en cours et attend
le worker avant de fermer SQLite.

Cette livraison se limite à la Phase 1. Les règles, quota quotidien, backoff,
rafraîchissement et expiration seront ajoutés en Phase 2/4 ; le score,
`ruleId`, `minScore`, `reviewStatus` et le dashboard de découverte en Phase 3.
Les lectures servent toujours les données persistées et ne les actualisent jamais.
