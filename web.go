package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

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

		tmpl := template.Must(template.New("index.html").Parse(`
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
					<th>Created At</th>
					<th>Currency</th>
					<th>Operate</th>
					<th>Amount</th>
					<th>Quantity</th>
					<th>Unit Price</th>
				</tr>
				{{range .Investments}}
				<tr>
					<td>{{.ID}}</td>
					<td>{{.CreatedAt}}</td>
					<td>{{.Currency}}</td>
					<td>{{.Operate}}</td>
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
					<th>Created At</th>
					<th>Date Form</th>
					<th>Currency</th>
					<th>Count</th>
					<th>Amount</th>
				</tr>
				{{range .Earns}}
				<tr>
					<td>{{.ID}}</td>
					<td>{{.CreatedAt}}</td>
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
