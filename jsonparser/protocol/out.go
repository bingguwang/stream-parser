package protocol

import (
	"fmt"
	"io/ioutil"
	"os/exec"
)

// GenerateHTML 生成HTML文件展示JSON数据
func GenerateHTML(outputFilePath string, jsonData []byte) error {
	htmlData := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Custom JSON Viewer</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #1e1e1e;
            color: #fff;
            margin: 0;
            padding: 20px;
        }
        #json-container {
            background-color: #333;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
        }
        .json-key {
            color: #72edf8;
        }
        .json-string {
            color: #aaffaa;
        }
        .json-number {
            color: #ff6666;
        }
        .json-boolean {
            color: #ffcc99;
        }
        .json-null {
            color: #cccccc;
        }
        .json-object-toggle {
            cursor: pointer;
            margin-right: 5px;
        }
        .json-object-hidden,
        .json-array-hidden {
            display: none;
        }
        ul {
            list-style-type: none;
            margin: 0;
            padding: 0;
        }
        li {
            margin-left: 20px;
            padding: 5px 0;
        }
        .json-array-item {
            margin-left: 20px;
            padding: 5px 0;
            cursor: pointer;
            border-bottom: 1px solid #444;
            transition: background-color 0.3s;
        }
        .json-array-length {
            color: #cccccc;
            margin-left: 5px;
        }
        .json-array-item:hover {
            background-color: #555;
        }
        .json-array-item.expanded {
            background-color: #444;
        }
        .json-array-toggle {
            cursor: pointer;
            margin-left: 5px;
        }
    </style>
</head>
<body>
    <div id="json-container"></div>

    <script>
        var container = document.getElementById("json-container");
        var jsonData = %s;

        function displayJSON(data, parentElement) {
            var type = typeof data;
            if (Array.isArray(data)) {
                var toggleButton = document.createElement("span");
                toggleButton.className = "json-array-toggle";
                toggleButton.textContent = "▶";
                parentElement.appendChild(toggleButton);

                var arrayLengthSpan = document.createElement("span");
                arrayLengthSpan.className = "json-array-length";
                arrayLengthSpan.textContent = "[" + data.length + "]";
                parentElement.appendChild(arrayLengthSpan);

                var ul = document.createElement("ul");
                ul.className = "json-array-hidden";
                parentElement.appendChild(ul);

                toggleButton.addEventListener("click", function() {
                    ul.classList.toggle("json-array-hidden");
                    toggleButton.textContent = ul.classList.contains("json-array-hidden") ? "▶" : "▼";
                });

                data.forEach(function(item) {
                    var li = document.createElement("li");
                    li.className = "json-array-item";
                    var toggleButton = document.createElement("span");
                    toggleButton.className = "json-object-toggle";
                    toggleButton.textContent = "▶";
                    li.appendChild(toggleButton);
                    var spanData = document.createElement("span");
                    if (typeof item === "string" || typeof item === 'number') {
                        spanData.textContent = item;
                    } else {
                        spanData.textContent = JSON.stringify(item.msg);
                    }
                    li.appendChild(spanData);
                    ul.appendChild(li);
                    var innerUl = document.createElement("ul");
                    innerUl.className = "json-object-hidden";
                    li.appendChild(innerUl);
                    toggleButton.addEventListener("click", function() {
                        innerUl.classList.toggle("json-object-hidden");
                        li.classList.toggle("expanded");
                        toggleButton.textContent = innerUl.classList.contains("json-object-hidden") ? "▶" : "▼";
                    });
                    displayJSON(item, innerUl);
                });
            } else if (type === "object" && data !== null) {
                var ul = document.createElement("ul");
                parentElement.appendChild(ul);
                for (var key in data) {
                    if (data.hasOwnProperty(key)) {
                        var li = document.createElement("li");
                        var spanKey = document.createElement("span");
                        spanKey.className = "json-key";
                        spanKey.textContent = key + ": ";
                        li.appendChild(spanKey);
                        ul.appendChild(li);
                        displayJSON(data[key], li);
                    }
                }
            } else {
                var span = document.createElement("span");
                switch (type) {
                    case "string":
                        span.className = "json-string";
                        break;
                    case "number":
                        span.className = "json-number";
                        break;
                    case "boolean":
                        span.className = "json-boolean";
                        break;
                    case "null":
                        span.className = "json-null";
                        break;
                }
                span.textContent = data;
                parentElement.appendChild(span);
            }
        }

        displayJSON(jsonData, container);
    </script>
</body>
</html>

`, string(jsonData))

	// 写入HTML文件
	if err := ioutil.WriteFile(outputFilePath+".html", []byte(htmlData), 0644); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/c", "start", outputFilePath+".html")

	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to open browser:", err)
		return err
	}

	return nil
}
