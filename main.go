package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mu sync.Mutex
var results [][]string
var uploadedFiles = map[string][]byte{}
var uploadedNames = map[string]string{}

func saveResult(d map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()
	row := []string{
		strconv.Itoa(len(results) + 1),
		fmt.Sprintf("%v", d["name"]),
		fmt.Sprintf("%v", d["id"]),
		fmt.Sprintf("%v", d["score"]),
		fmt.Sprintf("%v", d["grade"]),
		fmt.Sprintf("%v", d["correct"]),
		fmt.Sprintf("%v", d["wrong"]),
		fmt.Sprintf("%v", d["timeUsed"]),
		time.Now().Format("2006-01-02 15:04:05"),
	}
	results = append(results, row)
	fmt.Printf("✅ Saved: %v — %v%%\n", d["name"], d["score"])
}

func teacherPage() string {
	mu.Lock()
	rows := make([][]string, len(results))
	copy(rows, results)
	mu.Unlock()

	total := len(rows)
	var sum, hi, lo float64
	passed := 0
	if total > 0 { lo = 100 }
	for _, r := range rows {
		if len(r) < 4 { continue }
		s, _ := strconv.ParseFloat(r[3], 64)
		sum += s
		if s > hi { hi = s }
		if s < lo { lo = s }
		if s >= 50 { passed++ }
	}
	avg := 0.0
	if total > 0 { avg = sum / float64(total) }

	gc := map[string]string{"A+":"#059669","A":"#059669","B":"#2563eb","C":"#d97706","F":"#dc2626"}
	var rowsHTML strings.Builder
	if total == 0 {
		rowsHTML.WriteString("<tr><td colspan='9' style='text-align:center;padding:3rem;color:#94a3b8'>No submissions yet…</td></tr>")
	}
	for i, r := range rows {
		if len(r) < 9 { continue }
		sc, _ := strconv.ParseFloat(r[3], 64)
		scCol := "#059669"; if sc < 50 { scCol = "#dc2626" }
		bg := "#f8faff"; if i%2 != 0 { bg = "#fff" }
		g := strings.TrimSpace(strings.Split(r[4], "—")[0])
		g = strings.ReplaceAll(g, "+", "")
		col := gc[g]; if col == "" { col = "#dc2626" }
		rowsHTML.WriteString(fmt.Sprintf(`<tr style="background:%s">
<td style="padding:.7rem 1rem;color:#94a3b8;font-family:monospace">%s</td>
<td style="padding:.7rem 1rem;font-weight:600">%s</td>
<td style="padding:.7rem 1rem;font-family:monospace;color:#64748b">%s</td>
<td style="padding:.7rem 1rem;font-weight:700;font-size:1.05rem;color:%s">%s%%</td>
<td style="padding:.7rem 1rem"><span style="background:%s22;color:%s;padding:2px 10px;border-radius:99px;font-size:.78rem;font-weight:700;border:1px solid %s44">%s</span></td>
<td style="padding:.7rem 1rem;color:#059669;font-weight:600">%s</td>
<td style="padding:.7rem 1rem;color:#dc2626">%s</td>
<td style="padding:.7rem 1rem;color:#64748b;font-family:monospace">%s</td>
<td style="padding:.7rem 1rem;color:#94a3b8;font-size:.8rem">%s</td>
</tr>`, bg,r[0],r[1],r[2],scCol,r[3],col,col,col,r[4],r[5],r[6],r[7],r[8]))
	}

	var filesHTML strings.Builder
	mu.Lock()
	fnames := make([]string, 0, len(uploadedNames))
	for k := range uploadedNames { fnames = append(fnames, k) }
	mu.Unlock()
	if len(fnames) == 0 {
		filesHTML.WriteString("<p style='color:#94a3b8;text-align:center;padding:1.5rem'>No files uploaded yet</p>")
	}
	for _, k := range fnames {
		filesHTML.WriteString(fmt.Sprintf(`<div style="display:flex;align-items:center;gap:.8rem;padding:.6rem .8rem;border-radius:8px;background:#f8fafc;border:1px solid #e2e8f0;margin-bottom:.4rem">
<span>&#128196;</span>
<span style="flex:1;font-size:.84rem;font-family:monospace">%s</span>
<a href="/files/%s" download style="padding:.35rem .8rem;background:#dbeafe;color:#1e3a8a;border-radius:6px;font-size:.78rem;font-weight:600;text-decoration:none">&#8595; Download</a>
</div>`, uploadedNames[k], k))
	}

	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">
<meta http-equiv="refresh" content="10">
<title>Teacher Dashboard — DPU ITE 8.0</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{background:#f1f5f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#1e293b;min-height:100vh}
.hdr{background:linear-gradient(135deg,#1e3a8a,#2563eb);padding:1rem 2rem;display:flex;align-items:center;justify-content:space-between;box-shadow:0 2px 10px rgba(30,58,138,.35)}
.logo{display:flex;align-items:center;gap:12px}
.emb{width:46px;height:46px;background:#fff;border-radius:50%%;display:flex;flex-direction:column;align-items:center;justify-content:center;font-size:.72rem;font-weight:900;color:#1e3a8a}
h1{font-size:1.05rem;font-weight:700;color:#fff}
.sub{font-size:.7rem;color:#bfdbfe;display:block;margin-top:2px}
.live{background:#10b981;color:#fff;padding:.25rem .9rem;border-radius:99px;font-size:.72rem;font-weight:700}
.wrap{max-width:1200px;margin:0 auto;padding:1.5rem}
.stats{display:grid;grid-template-columns:repeat(6,1fr);gap:1rem;margin-bottom:1.5rem}
.s{background:#fff;border:1px solid #e2e8f0;border-radius:12px;padding:1.2rem;text-align:center;box-shadow:0 1px 3px rgba(0,0,0,.06)}
.sv{font-size:1.5rem;font-weight:700;font-family:monospace}
.sl{font-size:.63rem;color:#64748b;text-transform:uppercase;letter-spacing:.06em;margin-top:.3rem}
.card{background:#fff;border:1px solid #e2e8f0;border-radius:12px;padding:1.5rem;margin-bottom:1.5rem;box-shadow:0 1px 3px rgba(0,0,0,.06)}
.card-title{font-size:.9rem;font-weight:700;color:#1e293b;margin-bottom:1rem}
.acts{display:flex;gap:.8rem;align-items:center;margin-bottom:1.2rem;flex-wrap:wrap}
.btn{padding:.55rem 1.4rem;border-radius:8px;border:none;cursor:pointer;font-size:.85rem;font-weight:600;text-decoration:none;display:inline-block}
.b1{background:#dbeafe;color:#1e3a8a;border:1px solid #93c5fd}
.b2{background:#d1fae5;color:#065f46;border:1px solid #6ee7b7}
.note{font-size:.72rem;color:#64748b;font-family:monospace}
table{width:100%%;border-collapse:collapse;font-size:.86rem}
th{background:#1e3a8a;color:#fff;padding:.75rem 1rem;text-align:left;font-size:.67rem;text-transform:uppercase;letter-spacing:.08em}
tr:hover td{background:#eff6ff!important}
.ftr{background:#1e3a8a;padding:.65rem 2rem;display:flex;justify-content:space-between;margin-top:1rem}
.ftr span{font-size:.67rem;color:#93c5fd}
</style></head><body>
<div class="hdr">
  <div class="logo">
    <div class="emb"><div>DPU</div><div style="font-size:.45rem;color:#d97706">&#9733;&#9733;&#9733;</div></div>
    <div><h1>Teacher Dashboard — Module 2 Exam</h1>
    <span class="sub">Duhok Polytechnic University · ITE 8.0 · 2025-2026</span></div>
  </div>
  <span class="live">LIVE</span>
</div>
<div class="wrap">
  <div class="stats">
    <div class="s"><div class="sv" style="color:#1e40af">%d</div><div class="sl">Submitted</div></div>
    <div class="s"><div class="sv" style="color:#64748b">%d</div><div class="sl">Remaining</div></div>
    <div class="s"><div class="sv" style="color:#0891b2">%.1f%%</div><div class="sl">Class Avg</div></div>
    <div class="s"><div class="sv" style="color:#059669">%.0f%%</div><div class="sl">Highest</div></div>
    <div class="s"><div class="sv" style="color:#dc2626">%.0f%%</div><div class="sl">Lowest</div></div>
    <div class="s"><div class="sv" style="color:#059669">%d</div><div class="sl">Passed</div></div>
  </div>
  <div class="card">
    <div class="card-title">Student Results (%d submitted)</div>
    <div class="acts">
      <a href="/results.csv" class="btn b1" download>Download CSV</a>
      <button onclick="window.print()" class="btn b2">Print</button>
      <span class="note">Auto-refreshes every 10 seconds</span>
    </div>
    <div style="overflow-x:auto">
    <table><thead><tr>
      <th>#</th><th>Name</th><th>ID</th><th>Score</th><th>Grade</th>
      <th>Correct</th><th>Wrong</th><th>Time</th><th>Submitted At</th>
    </tr></thead><tbody>%s</tbody></table>
    </div>
  </div>
  <div class="card">
    <div class="card-title">Uploaded Packet Tracer Files (%d files)</div>
    %s
  </div>
</div>
<div class="ftr">
  <span>Duhok Polytechnic University — IT Department</span>
  <span>Last refresh: %s</span>
</div></body></html>`,
		total, 71-total, avg, hi, lo, passed, total,
		rowsHTML.String(), len(fnames), filesHTML.String(),
		time.Now().Format("15:04:05"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(examHTML))
	})

	http.HandleFunc("/teacher", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(teacherPage()))
	})

	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" { w.WriteHeader(200); return }
		var d map[string]interface{}
		json.NewDecoder(r.Body).Decode(&d)
		saveResult(d)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method != "POST" { return }
		ct := r.Header.Get("Content-Type")
		_, params, _ := mime.ParseMediaType(ct)
		mr := multipart.NewReader(r.Body, params["boundary"])
		var sName, sID string
		for {
			part, err := mr.NextPart()
			if err == io.EOF { break }
			if err != nil { break }
			switch part.FormName() {
			case "name": b, _ := io.ReadAll(part); sName = string(b)
			case "id":   b, _ := io.ReadAll(part); sID = string(b)
			case "file":
				orig := part.FileName()
				if orig == "" { continue }
				safe := sID + "_" + strings.ReplaceAll(sName, " ", "_") + "_" + orig
				safe = strings.Map(func(c rune) rune {
					if (c>='a'&&c<='z')||(c>='A'&&c<='Z')||(c>='0'&&c<='9')||c=='_'||c=='-'||c=='.' { return c }
					return '_'
				}, safe)
				data, _ := io.ReadAll(part)
				mu.Lock()
				uploadedFiles[safe] = data
				uploadedNames[safe] = orig
				mu.Unlock()
				fmt.Printf("File uploaded: %s\n", safe)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})

	http.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/files/")
		mu.Lock()
		data, ok := uploadedFiles[key]
		name := uploadedNames[key]
		mu.Unlock()
		if !ok { http.NotFound(w, r); return }
		w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(name))
		w.Write(data)
	})

	http.HandleFunc("/results.csv", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		rows := make([][]string, len(results))
		copy(rows, results)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=exam_results.csv")
		wr := csv.NewWriter(w)
		wr.Write([]string{"#","Name","ID","Score","Grade","Correct","Wrong","Time","Submitted"})
		wr.WriteAll(rows)
		wr.Flush()
	})

	fmt.Println("Server running on port", port)
	http.ListenAndServe(":"+port, nil)
}
