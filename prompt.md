# speckit-board : de zéro à la release v0.1.0

## GOAL

Livrer `speckit-board`, un kanban en terminal qui projette l'avancement des tâches SpecKit
(`specs/*/tasks.md`) sans base de données ni état propre, publié sur GitHub avec CI verte,
tag `v0.1.0` et release contenant les binaires Linux/macOS (amd64, arm64) et leurs checksums.

Le travail est terminé quand TOUS ces points sont vrais :
- `go install github.com/kOlapsis/speckit-board@latest` installe un binaire fonctionnel
- dans un repo SpecKit réel, `speckit-board` affiche le kanban et réagit en moins d'une
  seconde à une modification de `tasks.md`
- `speckit-board status` sort une ligne utilisable comme statusline Claude Code
- `speckit-board install-hooks` branche les hooks Claude Code de façon idempotente
- le workflow `ci.yml` est vert sur `main`, le workflow `release.yml` a produit la
  release `v0.1.0` avec les 4 binaires, `checksums.txt` et un changelog
- README (en + fr) permet d'installer et d'utiliser l'outil sans lire le code

## Contexte

- Je travaille en spec-driven avec GitHub Spec Kit et mon plugin speckit-guard
  (github.com/kOlapsis/speckit-guard) qui ajoute `/speckit-tests` et `/speckit-verify`.
- `tasks.md` contient des tâches `- [ ] T001 ...` / `- [x] T001 ...`, groupées par phases,
  avec marqueurs `[P]` pour le parallélisme. `/speckit-verify` appendit une section de
  tâches de remédiation à la fin de `tasks.md`. `acceptance-tests.md` contient une ligne
  `Reference: <SHA>` et une section « Spec ambiguities ».
- Je travaille dans tmux, Claude Code dans un pane, le board dans un pane à côté.
- Le board ne doit jamais être une source de vérité : s'il disparaît, rien n'est perdu.

## Décisions prises (ne pas rediscuter)

- Repo dédié `kOlapsis/speckit-board`, licence Apache 2.0, NOTICE comme speckit-guard.
- Go 1.23+, module `github.com/kOlapsis/speckit-board`, CGO désactivé, binaire statique.
- TUI : Bubble Tea + Lip Gloss + Bubbles. Surveillance fichiers : fsnotify. CLI : cobra.
- Convention « en cours » : `- [~] T007 ...` dans `tasks.md`. Le board la lit, il ne
  l'impose pas (c'est le rôle du prompt d'implémentation côté speckit-guard, hors périmètre).
- État des sessions Claude Code écrit par les hooks dans
  `$XDG_STATE_HOME/speckit-board/sessions/<session_id>.json` (défaut `~/.local/state`).
- Journal d'événements par projet dans `.specify/events.jsonl` (append-only, gitignoré
  par défaut via proposition dans le README, pas d'écriture dans `.gitignore` du projet).
- Releases avec GoReleaser, versionnement sémantique, commits Conventional Commits,
  changelog généré depuis les commits.

## Périmètre v0.1.0, dans cet ordre

### 1. Modèle et parsing
Parser `tasks.md` (tâches, phases, états `[ ]`/`[~]`/`[x]`, marqueur `[P]`, section de
remédiation) et `acceptance-tests.md` (ligne Reference, présence d'ambiguïtés non résolues).
Déduire les colonnes : Bloqué (phase précédente incomplète), Prêt, En cours, Fait,
Remédiation, Vérifié (critères OK du dernier verdict si trouvable dans les fichiers).
Tests unitaires sur fixtures réalistes (créer des fixtures plausibles, en incluant une
remédiation et la forme française `Référence :`). Le parser doit tolérer les variations de
format sans planter : une ligne non reconnue est ignorée, jamais fatale.

### 2. `speckit-board status`
Une ligne : feature courante (branche `NNN-name` ou `.specify/feature.json`), `fait/total`,
tâche en cours, dernier verdict verify si connu. Flag `--json` pour les intégrations.
Fournir dans le README l'extrait de configuration statusline Claude Code qui l'appelle.

### 3. TUI
- Kanban en colonnes, une swimlane par projet. Projets passés en arguments, sinon le
  répertoire courant. Rafraîchissement par fsnotify, repli sur polling si fsnotify échoue.
- Détail d'une tâche sur sélection (texte complet, phase, fichiers cités, horodatages
  issus de `events.jsonl` s'ils existent).
- Indicateur de session par swimlane : ambre = Claude attend une réponse ou une
  permission, vert = terminé, gris = inconnu. Source : fichiers de sessions des hooks.
- Touches : `j` saute au pane tmux de la session qui attend, `v` envoie `/speckit-verify`
  au pane Claude Code du projet, `Entrée` sur une tâche de remédiation envoie
  `/speckit-implement <id>`, `?` aide, `q` quitte. L'envoi passe par `tmux send-keys` ;
  hors tmux, les touches d'action affichent un message et ne font rien.
- Le pane cible est déduit de `tmux list-panes` (pane dont la commande est `claude` dans la
  même session, même répertoire) et surchargeable par `--target`.

### 4. Hooks Claude Code
- `speckit-board hook` lit le JSON du hook sur stdin (Notification, Stop, SubagentStop,
  PostToolUse sur Edit/Write/MultiEdit). Pour Notification/Stop : met à jour le fichier de
  session. Pour PostToolUse touchant `tasks.md` : calcule les transitions d'état et les
  appendit dans `events.jsonl` avec horodatage.
- `speckit-board install-hooks [--user|--project]` ajoute les entrées dans
  `settings.json` sans écraser les hooks existants, idempotent, avec `--dry-run`.
  Sortie 0 toujours : un hook qui échoue ne doit jamais bloquer Claude Code.
- Tests sur des payloads de hooks réels (format documenté par Anthropic).

### 5. Export HTML
`speckit-board --html out.html` écrit une page autonome (pas de CDN, pas de JS obligatoire)
avec la même projection, pour consultation sur téléphone.

## Hors périmètre v0.1.0
Serveur HTTP, notifications desktop, édition de `tasks.md` depuis le board,
réordonnancement, support Windows, intégration dans speckit-guard.

## Qualité et contraintes
- Pas d'état persistant propre au board autre que les sessions et `events.jsonl`.
- Un fichier `tasks.md` absent ou mal formé n'empêche jamais le board de démarrer.
- `golangci-lint` dans la CI, `go vet`, `go test -race`. Couverture du package de parsing
  d'au moins 85 %.
- Jamais de tiret cadratin dans les textes (README, aide CLI, commits).
- Code et commentaires en anglais, README en anglais avec `README.fr.md` traduit.
- Tests écrits avant l'implémentation pour chaque étape, commit « red » puis commit
  « green ». Un commit par tâche, messages Conventional Commits.

## CI et release
- `.github/workflows/ci.yml` : sur push et pull request, lint + tests + build matrice
  (linux/darwin × amd64/arm64), CGO désactivé, cache des modules.
- `.github/workflows/release.yml` : sur tag `v*`, vérifie que le tag est un semver, lance
  les tests, puis GoReleaser : archives tar.gz, `checksums.txt`, changelog depuis les
  commits, binaires attachés à la release GitHub. `GITHUB_TOKEN` suffit, pas de secret
  supplémentaire.
- `.goreleaser.yaml` avec `ldflags` injectant version, commit et date ; `speckit-board
  version` les affiche.
- Dependabot pour les modules Go et les actions GitHub, hebdomadaire.
- `CHANGELOG.md` initial, `.editorconfig`, `Makefile` avec `build`, `test`, `lint`,
  `snapshot` (GoReleaser local sans publication).

## Méthode de travail
- Ne pose une question que si une décision est irréversible et réellement ambiguë ;
  sinon tranche, note le choix dans `docs/DECISIONS.md` et continue.
- Après chaque étape du périmètre : tests verts, lint vert, commit, puis étape suivante.
- Avant de taguer : `goreleaser release --snapshot --clean` doit réussir localement,
  et `gh workflow run` ou un push doit montrer `ci.yml` vert.
- Fin de travail : créer le repo distant avec `gh repo create kOlapsis/speckit-board
  --public`, pousser `main`, taguer `v0.1.0` sur la tête de `main`, pousser le tag,
  suivre `release.yml` avec `gh run watch` jusqu'au succès, puis vérifier avec
  `gh release view v0.1.0` que les 4 archives et `checksums.txt` sont présents.
- Rapport final : ce qui est livré, les décisions prises, les limites connues, et la
  commande d'installation. Rien d'autre.