# YouTube Cache Proxy

Service Go en cours de développement pour mettre en cache l'audio de vidéos YouTube.

## État actuel

Le dépôt contient les fondations du backend :

- un serveur HTTP avec arrêt propre sur `SIGINT` et `SIGTERM` ;
- une API REST pour ajouter, filtrer, consulter et supprimer des sources, lancer un lot manuel, relancer une demande échouée, servir l'audio et contrôler le worker ;
- `GET /health`, qui renvoie `{"status":"ok"}` ;
- une configuration par variables d'environnement, avec validation des limites de téléchargement et de reprise ;
- un stockage SQLite avec migrations et opérations de file d'attente ;
- une suppression atomique des sources, refusée tant qu'elles appartiennent à un lot actif ;
- un composant interne d'extraction des métadonnées et de téléchargement audio via `yt-dlp`, avec limites de durée, taille, débit et délai d'exécution ;
- un moteur interne qui réserve un lot manuel, traite les téléchargements en arrière-plan, respecte le quota glissant sur 24 h et attend une pause aléatoire entre tentatives ;
- une classification des erreurs `yt-dlp` pour distinguer les échecs réseau relançables des erreurs définitives.

**L'API de base est maintenant utilisable, mais le projet reste en développement.** Les routes REST listées ci-dessous sont disponibles, notamment la récupération des fichiers audio. L'authentification Bearer et les erreurs JSON uniformes restent à finaliser ; ne l'expose pas sur un réseau non fiable. Il n'y a pas encore d'interface utilisateur ni de configuration Docker Compose dans le dépôt.

L'API accepte temporairement les demandes sans authentification jusqu'à l'issue dédiée aux rôles Bearer. L'endpoint audio utilise le service de fichiers Go, qui prend en charge les requêtes Range.

## API REST actuelle

| Méthode | Endpoint | Description |
|---|---|---|
| `POST` | `/sources` | Ajouter une URL YouTube (`{"url":"https://..."}`) |
| `GET` | `/sources?status=pending&limit=50&offset=0` | Lister et filtrer les sources |
| `GET` | `/sources/{id}` | Lire une demande |
| `GET` | `/sources/{id}/audio` | Récupérer le fichier audio prêt, avec support Range natif |
| `POST` | `/downloads/run` | Démarrer un lot asynchrone (`{"count":3}`) |
| `POST` | `/sources/{id}/retry` | Remettre une demande échouée en attente |
| `DELETE` | `/sources/{id}` | Supprimer une demande et son fichier audio |
| `POST` | `/worker/pause` | Empêcher le démarrage de la prochaine vidéo |
| `POST` | `/worker/resume` | Reprendre un lot actif |
| `GET` | `/worker/status` | Lire les compteurs et l'état du worker |
| `GET` | `/health` | Vérifier la réponse HTTP |

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

Le binaire `yt-dlp` doit être installé et disponible dans `PATH` pour utiliser le composant interne de téléchargement. Ce composant n'est pas encore accessible via une route HTTP.

## Vérifications

```sh
go test ./...
go vet ./...
```
