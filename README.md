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

**L'API de base est maintenant utilisable, mais le projet reste en développement.** Les routes REST listées ci-dessous sont disponibles, notamment la récupération des fichiers audio. Les erreurs JSON uniformes restent à finaliser ; ne l'expose pas sur un réseau non fiable. Il n'y a pas encore d'interface utilisateur ni de configuration Docker Compose dans le dépôt.

L'API exige un jeton Bearer : le rôle `producer` peut ajouter et consulter les sources et récupérer l'audio ; le rôle `admin` peut aussi démarrer des lots, relancer ou supprimer une source et contrôler le worker. `/health` reste public. L'endpoint audio utilise le service de fichiers Go, qui prend en charge les requêtes Range.

La pause laisse la vidéo en cours se terminer, puis conserve le lot actif pour la reprise. Au redémarrage du service, un lot actif est marqué `interrupted` et ses sources inachevées redeviennent `pending` ; aucun téléchargement ne reprend sans nouveau lancement explicite.

`POST /sources` lance une requête yt-dlp limitée aux métadonnées (30 secondes maximum), sans téléchargement audio. Si YouTube ou yt-dlp ne renvoie pas les détails, la source reste créée en `pending` et les champs indisponibles sont omis. Les métadonnées enrichies apparaissent également dans les réponses de liste et de détail.

Configure `PRODUCER_TOKEN` et `ADMIN_TOKEN` avec deux valeurs différentes d'au moins 32 caractères. Les deux variables sont obligatoires au démarrage.

## API REST actuelle

| Méthode | Endpoint | Description |
|---|---|---|
| `POST` | `/sources` | Producer/Admin : ajouter une URL YouTube et récupérer ses métadonnées (`{"url":"https://..."}`) |
| `GET` | `/sources?status=pending&limit=50&offset=0` | Producer/Admin : lister et filtrer les sources avec leurs métadonnées |
| `GET` | `/sources/{id}` | Producer/Admin : lire une demande et ses métadonnées |
| `GET` | `/sources/{id}/audio` | Producer/Admin : récupérer l'audio prêt, avec support Range natif |
| `POST` | `/downloads/run` | Admin : démarrer un lot asynchrone (`{"count":3}`) |
| `POST` | `/sources/{id}/retry` | Admin : remettre une demande échouée en attente |
| `DELETE` | `/sources/{id}` | Admin : supprimer une demande et son fichier audio |
| `POST` | `/worker/pause` | Admin : empêcher le démarrage de la prochaine vidéo |
| `POST` | `/worker/resume` | Admin : reprendre un lot actif |
| `GET` | `/worker/status` | Admin : lire les compteurs et l'état du worker |
| `GET` | `/health` | Public : vérifier la réponse HTTP |

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

Le binaire `yt-dlp` doit être installé et disponible dans `PATH` pour traiter les lots démarrés via l'API.

## Vérifications

```sh
go test ./...
go vet ./...
```
