package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// Define a simple webpage to display the color information
	tmpl := template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.0.2/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-EVSTQN3/azprG1Anm3QDgpJLIm9Nao0Yz1ztcQTwFspd3yD65VohhpuuCOmLASjC" crossorigin="anonymous">
<head>
<title>My Color Application</title>
</head>
<body>

<nav class="navbar navbar-light bg-light">
  <div class="container-fluid">
    <span class="navbar-brand mb-0 h1">My Color Application</span>
  </div>
</nav>
<div class="container">
  <div class="row align-items-start">
    <div class="col-4">
		<table class="table table-bordered">
		<thead>
		<tr>
		<th>Name</th>
		<th>Value</th>
		</tr>
		</thead>
		<tbody>
		{{range .ConfigValues}}
		<tr>
			<td>{{.Name}}</td>
			<td>{{.Value}}</td>
		</tr>
		{{end}}
		</tbody>
		</table>
    </div>


    <div class="col">
		<h1>API Request Value Stream</h1>
		<table class="table">
		<thead>
		<tr>
		<th>Time</th>
		<th>Name</th>
		<th>Color</th>
		</tr>
		</thead>
		<tbody id="apiTable">
		</tbody>
		</table>
	</div>
</div>

<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.0.2/dist/js/bootstrap.bundle.min.js" integrity="sha384-MrcW6ZMFYlzcLA8Nl+NtUVF0sA7MsXsP1UyJoMp4YLEuNSfAP+JcXn/tWtIaxVXM" crossorigin="anonymous"></script>
<script>
setInterval(function() {
    // Get the data from the API endpoint.
    var xhr = new XMLHttpRequest();
    xhr.open("GET", "/api/data");
    xhr.onload = function() {
        // Parse the JSON response.
        var data = JSON.parse(xhr.responseText);
        // Append the data to the table.
        for (var i = 0; i < data.length; i++) {
            var row = document.createElement("tr");
            row.innerHTML = ` + "`<td>${data[i].time}</td> <td>${data[i].name}</td> <td style=\"background-color:${data[i].color}\">${data[i].color}</td>`;" + `
            document.getElementById("apiTable").prepend(row);
        }
    };
    xhr.send();
}, 1000);
</script>
</body>
</html>
`))
	hostname := os.Getenv("HOSTNAME")
	remoteColorService := os.Getenv("AppClrScv")

	// Define a handler to return the webpage
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Render the template.
		tmpl.Execute(w, TemplateModel{ConfigValues: GetAppValues()})
	})

	// Define the route to return the color data queried by the website
	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		if remoteColorService == "" {
			ReturnColorData(hostname, "red", w)
		} else {
			name, color, err := getColorName("http://" + remoteColorService + "/color")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			ReturnColorData(name, color, w)
		}
	})

	// Listen on port 8080.
	http.ListenAndServe(":8080", nil)
}

type NameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TemplateModel struct {
	ConfigValues []NameValue
}

// GetAppValues returns the 'App Values' which are all env vars that start with the string 'App'
func GetAppValues() []NameValue {
	var result []NameValue
	for _, keyValueStr := range os.Environ() {
		keyValue := strings.SplitN(keyValueStr, "=", 2)
		key := keyValue[0]
		value := keyValue[1]
		idx := strings.Index(key, "App")
		if idx >= 0 {
			result = append(result, NameValue{Name: key[idx+3:], Value: value})
		}
	}

	return result
}

// ReturnColorData writes the provided color data to the ResponseWriter
func ReturnColorData(name string, color string, w http.ResponseWriter) {
	people := []struct {
		Name  string `json:"name"`
		Time  string `json:"time"`
		Color string `json:"color"`
	}{
		{name, time.Now().Format("2006-01-02 15:04:05"), color},
	}
	json.NewEncoder(w).Encode(people)
}

// getColorName gets a color from the backend
func getColorName(endpoint string) (string, string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", "", err
	}
	req.Close = true
	response, err := client.Do(req)
	if err != nil {
		return "", "", err
	}

	if response.StatusCode != 200 {
		return "", "", fmt.Errorf("Error getting response: %d", response.StatusCode)
	}

	var data struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", "", err
	}

	return data.Name, data.Color, nil
}
