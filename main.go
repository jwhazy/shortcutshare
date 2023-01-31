package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sqweek/dialog"
	"github.com/urfave/cli/v2"
)

var port int
var file, directory string
var no_upload, no_serve, no_save bool

var config Config

type Config struct {
	Port      int    `json:"port"`
	Directory string `json:"directory"`
}

func main() {
	fmt.Println("\nShortcutShare v0.0.0")

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "port",
				Aliases:     []string{"p"},
				Usage:       "Sets the port to listen on",
				Destination: &port,
			},

			&cli.StringFlag{
				Name:        "file",
				Aliases:     []string{"f"},
				Usage:       "Sets the port to serve",
				Destination: &file,
			},

			&cli.StringFlag{
				Name:        "directory",
				Aliases:     []string{"d"},
				Usage:       "Directory to save file to.",
				Destination: &directory,
			},

			&cli.BoolFlag{
				Name:        "no-serve",
				Aliases:     []string{"x"},
				Usage:       "Do not serve any files, ignores file picker.",
				Value:       false,
				Destination: &no_serve,
			},

			&cli.BoolFlag{
				Name:        "no-upload",
				Aliases:     []string{"u"},
				Usage:       "Do not allow uploading via POST.",
				Value:       false,
				Destination: &no_upload,
			},

			&cli.BoolFlag{
				Name:        "no-save",
				Aliases:     []string{"s"},
				Usage:       "Do not save passed in arguments to config.",
				Value:       false,
				Destination: &no_save,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if file == "" {
				cwd, err := os.Getwd()

				if err != nil {
					cwd = "C:\\"
				}

				dialog, err := dialog.File().SetStartDir(cwd).Load()

				if err != nil || dialog == "" || no_serve {
					log.Println("No file specified. Upload only enabled.")
					no_serve = true
				} else {
					file = dialog
					log.Println("Serving file: ", file)
				}

			}

			parse_config()

			if directory != "" || no_upload {
				log.Println("Uploading to: ", directory)
			}

			if !no_save {
				save_config()
			} else {
				log.Println("Saving config disabled.")
			}

			log.Println("Listening on port: ", port)

			setup_http()

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func save_config() {
	config.Port = port
	config.Directory = directory

	file, _ := json.MarshalIndent(config, "", " ")

	_ = os.WriteFile("config.json", file, 0644)
}

func parse_config() {
	cwd, err := os.Getwd()

	if err != nil {
		log.Println("Config could not be created, reason: ", err)
	}

	if _, err := os.Stat(filepath.Join(cwd, "config.json")); err == nil {
		file, _ := os.ReadFile("config.json")

		json.Unmarshal(file, &config)
	} else {
		log.Println("No config file found, creating.")

		if port == 0 {
			port = 3000
		}

		if directory == "" {
			directory = "C:\\ShortcutShare"
		}

		return
	}

	if port == 0 {
		if config.Port == 0 {
			port = 3000
		} else {
			port = config.Port
		}
	}

	if directory == "" {
		if config.Directory == "" {
			directory = "C:\\ShortcutShare"
		} else {
			directory = config.Directory
		}
	}
}

func setup_http() {
	if no_upload {
		http.HandleFunc("/upload", uploadFile)
	}

	if !no_serve || file != "" {
		http.HandleFunc("/get", getFile)

	}

	http.HandleFunc("/", func(req http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/" {
			req.WriteHeader(404)
			return
		}
	})
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func getFile(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		res.WriteHeader(405)
		fmt.Sprintln(res, "Invalid request method.")
		return
	}

	res.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(file))
	http.ServeFile(res, req, file)
}

func uploadFile(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		res.WriteHeader(405)
		fmt.Sprintln(res, "Invalid request method.")
		return
	}

	req.ParseMultipartForm(32 << 20)
	input, handler, err := req.FormFile("file")

	if err != nil {
		fmt.Println(err)
		http.Error(res, "Please upload using multipart/form-data, make sure the key is `file`.", 400)
		return
	}

	defer input.Close()

	dst, err := os.Create(filepath.Join(directory, handler.Filename))

	if err != nil {
		fmt.Println(err)
		http.Error(res, err.Error(), 500)
		return
	}

	defer dst.Close()

	if _, err := io.Copy(dst, input); err != nil {
		http.Error(res, err.Error(), 500)
		return
	}

	fmt.Sprintln(res, "File successfully uploaded.")
}
