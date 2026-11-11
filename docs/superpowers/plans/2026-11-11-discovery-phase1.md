# Discovery Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans.

**Goal:** Rechercher une fois et relire le cache sans appel YouTube.
**Architecture:** tables additives dans store; client et worker discovery; routes httpapi.
**Tech Stack:** Go 1.27, net/http, SQLite existant.
**Spec:** docs/superpowers/specs/2026-11-11-discovery-design.md

## Global Constraints
Authentification existante; aucune modification fonctionnelle des téléchargements.
Pas de dépendance frontend, concurrence 1, aucun appel YouTube dans les lectures.

## Review Focus
Erreurs API, arrêt pendant un appel, résultats dupliqués, requêtes invalides,
lecture concurrente à une recherche.

### Task 1: Persistance
- [x] Tests puis migration 0004 et opérations dans internal/store/discovery.go.
- [x] Vérifier déduplication, pagination, transaction et reprise après redémarrage.

### Task 2: Exécution
- [x] Tests avec httptest puis client Data API et worker à notification.
- [x] Vérifier paramètres, erreurs et absence d'attente HTTP.

### Task 3: API et intégration
- [x] Tests auth et cache puis routes, branchement main et documentation.
- [x] go test ./..., go test -race ./..., go vet ./...

## Vérification et revue
Revue indépendante effectuée : deux défauts corrigés avec régressions
(TestDiscoveryDatesCompareInstants et TestDiscoveryPersistence).
Suite standard et go vet : succès. Race discovery/httpapi/store : succès.
Race global : trois échecs préexistants reproduits sur dev inchangée dans
TestDownloadRejectsMetadataOutsideLimitsBeforeTransfer (duration, known_filesize,
unknown_duration), chaque sous-test imposant un timeout de 1 seconde au sous-processus.
Aucune modification du téléchargement pour contourner ces tests.
Décisions : une page de 50 résultats; concurrence 1; aucune reprise automatique
des appels interrompus; filtres de règles/scoring et quotas différés aux phases prévues.
