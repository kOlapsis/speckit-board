package export

import (
	"fmt"
	"html/template"
	"io"

	"github.com/kOlapsis/speckit-board/internal/sessions"
	"github.com/kOlapsis/speckit-board/internal/spec"
)

type Lane struct {
	Project spec.Project
	Light   sessions.Light
}

type cardView struct {
	ID       string
	Text     string
	Parallel bool
}

type columnView struct {
	ID    string
	Title string
	Cards []cardView
}

type laneView struct {
	Name, Feature, Verdict string
	Done, Total            int
	Light, LightLabel      string
	HasTasks               bool
	Columns                []columnView
}

var lightNames = map[sessions.Light][2]string{
	sessions.Amber: {"amber", "waiting"},
	sessions.Green: {"green", "done"},
	sessions.Gray:  {"gray", "unknown"},
}

func HTML(w io.Writer, lanes []Lane) error {
	views := make([]laneView, len(lanes))
	for i, l := range lanes {
		done, total := l.Project.Counts()
		names := lightNames[l.Light]
		v := laneView{
			Name: l.Project.Name, Feature: l.Project.Feature, Verdict: l.Project.Verification.Verdict,
			Done: done, Total: total, Light: names[0], LightLabel: names[1], HasTasks: l.Project.HasTasks,
		}
		for _, c := range l.Project.Columns() {
			cv := columnView{ID: string(c.ID), Title: c.ID.Title()}
			for _, card := range c.Cards {
				cd := cardView{ID: card.ID}
				if card.Task != nil {
					cd.Text, cd.Parallel = card.Title, card.Task.Parallel
				}
				cv.Cards = append(cv.Cards, cd)
			}
			v.Columns = append(v.Columns, cv)
		}
		views[i] = v
	}
	if err := page.Execute(w, views); err != nil {
		return fmt.Errorf("render html: %w", err)
	}
	return nil
}

var page = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>speckit-board</title>
<style>
:root{--bg:#f6f6f4;--fg:#1c1c1a;--card:#fff;--line:#d9d9d4;--dim:#6b6b66;--amber:#c27c00;--green:#2a8a4a;--gray:#8a8a85}
@media (prefers-color-scheme:dark){:root{--bg:#161614;--fg:#ececE8;--card:#222220;--line:#3a3a36;--dim:#9a9a94;--amber:#f0a830;--green:#4fc070;--gray:#8a8a85}}
*{box-sizing:border-box}
body{margin:0;padding:16px;background:var(--bg);color:var(--fg);font:15px/1.4 system-ui,sans-serif}
h1{font-size:18px;margin:0 0 16px}
section{margin:0 0 28px}
.head{display:flex;flex-wrap:wrap;gap:4px 14px;align-items:baseline;margin:0 0 10px}
.head b{font-size:17px}
.head span{color:var(--dim)}
.dot{font-weight:600}
.dot.amber{color:var(--amber)}.dot.green{color:var(--green)}.dot.gray{color:var(--gray)}
.cols{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:10px}
.col{background:var(--card);border:1px solid var(--line);border-radius:6px;padding:8px}
.col h3{font-size:13px;text-transform:uppercase;letter-spacing:.04em;margin:0 0 6px;color:var(--dim)}
.col ul{list-style:none;margin:0;padding:0}
.col li{padding:5px 0;border-top:1px solid var(--line);overflow-wrap:anywhere}
.col li:first-child{border-top:0}
.id{font-weight:600;margin-right:6px}
.p{font-size:11px;border:1px solid var(--line);border-radius:3px;padding:0 3px;color:var(--dim);margin-left:4px}
.empty{color:var(--dim)}
</style>
</head>
<body>
<h1>speckit-board</h1>
{{range .}}<section>
<div class="head"><b>{{.Name}}</b><span>{{if .Feature}}{{.Feature}}{{else}}no feature{{end}}</span><span>{{.Done}}/{{.Total}}</span>{{if .Verdict}}<span>verify {{.Verdict}}</span>{{end}}<span class="dot {{.Light}}">&#9679; {{.LightLabel}}</span></div>
{{if .HasTasks}}<div class="cols">{{range .Columns}}<div class="col col-{{.ID}}"><h3>{{.Title}} ({{len .Cards}})</h3>{{if .Cards}}<ul>{{range .Cards}}<li><span class="id">{{.ID}}</span>{{.Text}}{{if .Parallel}}<span class="p">P</span>{{end}}</li>{{end}}</ul>{{end}}</div>{{end}}</div>
{{else}}<p class="empty">no tasks.md found</p>{{end}}</section>
{{end}}</body>
</html>
`))
