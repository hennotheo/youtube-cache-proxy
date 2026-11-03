# YouTube Cache Proxy

Service Go en cours de développement pour mettre en cache l'audio de vidéos YouTube.

## État actuel

Le dépôt contient les fondations du backend :

- un serveur HTTP avec arrêt propre sur `SIGINT` et `SIGTERM` ;
- `GET /health`, qui renvoie `{"status":"ok"}` ;
- une configuration par variables d'environnement, avec validation des limites de téléchargement et de reprise ;
- un stockage SQLite avec migrations et opérations de file d'attente ;
- une suppression atomique des sources, refusée tant qu'elles appartiennent à un lot actif ;
- un composant interne d'extraction des métadonnées et de téléchargement audio via `yt-dlp`, avec limites de durée, taille, débit et délai d'exécution ;
- un moteur interne qui réserve un lot manuel puis traite ses téléchargements en arrière-plan ;
- une classification des erreurs `yt-dlp` pour distinguer les échecs réseau relançables des erreurs définitives.

**L'application n'est pas encore utilisable comme proxy audio complet.** À ce stade, seule la route de santé est exposée par HTTP : il n'y a pas encore d'API publique pour ajouter ou récupérer des sources, consulter la file ou lire les fichiers audio. Il n'y a pas non plus d'interface utilisateur ni de configuration Docker Compose dans le dépôt.

Le moteur de lots est une capacité interne, pas encore déclenchable par une route HTTP. L'API REST sera exposée dans une issue ultérieure.

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
