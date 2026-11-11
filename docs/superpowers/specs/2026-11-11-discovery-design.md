# Cache de découverte — Phase 1
Extension incrémentale de dev : conserver sources, téléchargements, authentification et dashboard.
Ajouter un cache indépendant dans la même base SQLite WAL : recherches persistantes,
vidéos uniques par videoId et associations recherche/vidéo.
Une recherche admin répond 202 après insertion SQLite. Un seul worker consomme la file,
sans polling (notification et vidage au démarrage). Les lectures Producer/Admin utilisent
uniquement SQLite. Une exécution interrompue est marquée interrupted au redémarrage,
sans répéter automatiquement un appel potentiellement facturé.
YouTube Data API : une page search.list (50 résultats maximum), puis videos.list groupé
pour les détails et durées. Client HTTP réutilisé, timeout, erreurs publiques expurgées.
Clé facultative YOUTUBE_API_KEY : lectures possibles sans clé, lancement indisponible.
Alternatives : seconde base écartée (cycle de vie supplémentaire), réutilisation de sources
écartée (une découverte ne doit pas réserver un téléchargement).
Phase 2 : règles, quota quotidien, backoff et planification. Phase 3 : score et validation,
dashboard. Phase 4 : actualisation intelligente, statistiques et optimisations.
