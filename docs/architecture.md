# Football AI Predictor - Architecture

## Flux de prédiction

1. React demande une analyse via Go API
2. Go charge les features depuis PostgreSQL
3. Go appelle Flask ML Service pour les probabilités
4. Go appelle AI Agent pour l'explication
5. Résultat persisté et retourné au frontend

## Anti-leakage

Toutes les features utilisent uniquement des données antérieures au `kickoff_at` du match.
Chaque prédiction enregistre `data_snapshot_at`.

## Modèles ML V1

| Modèle | Marchés |
|--------|---------|
| 1X2 | Home, Draw, Away, Double chance |
| Goals | O/U 1.5, 2.5, 3.5, BTTS |
| Corners | O/U corners |
| Shots | O/U tirs, tirs cadrés |

## Split temporel

- 2018-2022 : Training
- 2023 : Validation
- 2024 : Test
- 2025-2026 : Backtesting réel
