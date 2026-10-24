# speckit-board

Un kanban en terminal pour les projets [GitHub Spec Kit](https://github.com/github/spec-kit). Il lit
`specs/*/tasks.md` et montre où en est chaque tâche, à côté du pane tmux où tourne Claude Code.

Le board ne possède aucune donnée. Il projette seulement les fichiers de votre dépôt : s'il disparaît,
rien n'est perdu.

## Installation

```sh
go install github.com/kOlapsis/speckit-board@latest
```

Ou téléchargez une archive Linux ou macOS (amd64, arm64) sur la
[page des releases](https://github.com/kOlapsis/speckit-board/releases) et vérifiez-la avec `checksums.txt`.

## Utiliser le board

```sh
speckit-board                  # répertoire courant
speckit-board ~/work/app ~/work/api   # une swimlane par projet
```

Colonnes :

| Colonne | Signification |
|---------|---------------|
| Blocked | à faire, mais une phase précédente a encore des tâches non terminées |
| Ready | à faire, rien ne la bloque |
| In progress | `- [~] T007 ...` |
| Done | `- [x] T007 ...` |
| Remediation | tâches ouvertes de la section « Remediation (verify) » que `/speckit-verify` ajoute |
| Verified | critères notés OK dans le `verification.md` de la feature |

`[ ]`, `[~]` et `[x]` sont lus tels quels. Marquer une tâche `[~]` pendant qu'on la traite est une
convention à mettre dans votre prompt d'implémentation. Le board ne l'écrit jamais.

La feature courante est la branche `NNN-name`, sinon `.specify/feature.json`, sinon la feature dont le
`tasks.md` a changé en dernier. Un `tasks.md` absent ou mal formé n'empêche jamais le board de démarrer, les
lignes illisibles sont ignorées.

Le board se met à jour en moins d'une seconde quand un fichier change (fsnotify, avec repli sur du polling).

### Touches

| Touche | Action |
|--------|--------|
| flèches, `h`, `l` | changer de colonne, flèches haut et bas pour changer de carte |
| `tab` | projet suivant |
| `j` | aller au pane tmux de la session Claude Code qui attend |
| `v` | envoyer `/speckit-verify` au pane Claude Code du projet |
| `entrée` | sur une tâche de remédiation, envoyer `/speckit-implement <id>` |
| `?` | aide |
| `q` | quitter |

Les actions passent par `tmux send-keys`. Hors tmux, elles affichent seulement un message. Le pane visé est
celui qui exécute `claude` dans la même session tmux et le même répertoire que le projet. On le change avec
`--target %3`.

Chaque swimlane affiche un voyant de session : ambre quand Claude attend une réponse ou une permission, vert
quand il a terminé, gris quand c'est inconnu. Il vient des hooks décrits plus bas.

## Statusline de Claude Code

```sh
speckit-board status          # 003-auth 7/12 · ▶ T008 Add login handler · verify PASS
speckit-board status --json
```

Dans `~/.claude/settings.json` :

```json
{
  "statusLine": { "type": "command", "command": "speckit-board status" }
}
```

## Hooks

```sh
speckit-board install-hooks --dry-run   # affiche le résultat
speckit-board install-hooks --project   # ./.claude/settings.json (par défaut)
speckit-board install-hooks --user      # ~/.claude/settings.json
```

La commande ajoute des entrées Notification, Stop et PostToolUse qui lancent `speckit-board hook`, garde vos
hooks existants et peut être relancée sans risque. Le binaire doit être dans votre `PATH`. La commande `hook`
sort toujours avec le code 0 : elle ne peut jamais bloquer Claude Code.

Les hooks écrivent deux choses :

- `$XDG_STATE_HOME/speckit-board/sessions/<session_id>.json` (par défaut `~/.local/state`), pour les voyants de session ;
- `.specify/events.jsonl` dans le projet, un journal en ajout seul des changements d'état des tâches, qui donne
  leurs heures de début et de fin au panneau de détail. Ajoutez-le à votre `.gitignore` : le board ne modifie
  jamais ce fichier à votre place.

```
.specify/events.jsonl
```

## Export HTML

```sh
speckit-board --html board.html ~/work/app
```

Écrit une page autonome (sans script ni ressource externe), lisible sur téléphone.

## Compilation

```sh
make build test lint snapshot
```

Sous licence Apache 2.0.
