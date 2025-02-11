package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/simon4545/binance-macd/db"
)

func WebInit() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var investments []db.Investment
		var earns []db.Earn

		investments = db.GetAllInvestments()
		earns = db.GetAllEarns()

		data := struct {
			Investments []db.Investment
			Earns       []db.Earn
		}{
			Investments: investments,
			Earns:       earns,
		}
		// 创建模板函数，用于格式化时间
		funcMap := template.FuncMap{
			"formatTime": func(t time.Time) string {
				return t.Format("2006-01-02 15:04:05")
			},
		}

		tmpl := template.Must(template.New("index.html").Funcs(funcMap).Parse(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Investments and Earns</title>
		</head>
		<body>
			<h1>Investments</h1>
			<table border="1">
				<tr>
					<th>ID</th>
					<th>时间</th>
					<th>代币</th>
					<th>金额</th>
					<th>数量</th>
					<th>单价</th>
				</tr>
				{{range .Investments}}
				<tr>
					<td>{{.ID}}</td>
					<td>{{.CreatedAt | formatTime}}</td>
					<td>{{.Currency}}</td>
					<td>{{.Amount}}</td>
					<td>{{.Quantity}}</td>
					<td>{{.UnitPrice}}</td>
				</tr>
				{{end}}
			</table>

			<h1>Earns</h1>
			<table border="1">
				<tr>
					<th>ID</th>
					<th>日期</th>
					<th>代币</th>
					<th>成交次数</th>
					<th>收益</th>
				</tr>
				{{range .Earns}}
				<tr>
					<td>{{.ID}}</td>
					<td>{{.DateForm}}</td>
					<td>{{.Currency}}</td>
					<td>{{.Count}}</td>
					<td>{{.Amount}}</td>
				</tr>
				{{end}}
			</table>
		</body>
		</html>
		`))

		tmpl.Execute(w, data)
	})

	fmt.Println(fmt.Sprintf("Server is running on http://localhost:%d", config.Port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}
