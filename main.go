package main

import (
	"crypto/sha512"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful/v3"
	_ "github.com/lib/pq"
	"golang.org/x/exp/slices"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type User struct {
	DiscordID string `json:"discordID"`
}

type HashedUser struct {
	UserHashID string `json:"userHashID"`
}

type TerminalCMD struct {
	Location string `json:"loc"`
	Command  string `json:"cmd"`
	User     string `json:"user"`
}

type TerminalResult struct {
	Message   string `json:"message"`
	NewFolder string `json:"new_folder"`
	NewUser   string `json:"new_user"`
	Redirect  string `json:"redirect"`
}

type Games struct {
	UserHashID string `json:"userHashID"`
	DRG        bool   `json:"DRG"`
	HFF        bool   `json:"HFF"`
	LEFT       bool   `json:"LEFT"`
	SOW        bool   `json:"SOW	"`
}

const (
	userTable  = "users"
	gamesTable = "games"
	userId     = "discord_id"
	userHash   = "hashed_id"
	grinchTxt  = "Grinch here,\nGot a new idea for stopping Christmas, just hack into Santa's computer.\nTurns out instead of hashing & salting his passwords, he turns them \ninto cookies then adds milk.\n\nI uploaded a small program that can find his password, you just need to\ngive it one word from the password and it will find the rest.\nTo run it just type `raisins_no_choco <guess>`.\n\nYou should look into /santa_secrets/ and see what you can find."
	santaTxt   = "I love my reindeers a lot,\nBut the one I love most,\nIs the one with the red nose!\n"
)

func files(folder string, user string) []string {
	if folder == "~" && user == "guest" {
		return []string{"readme.txt"}
	} else if folder == "~" && user == "santa" {
		return []string{"invitation.jpg"}
	} else if folder == "/santa_secrets/" {
		return []string{"reminder.txt"}
	}
	return []string{}
}

func cmdList() []string {
	return []string{"ls", "cat", "cd", "raisins_no_choco", "login", "open"}
}

func cmdHelp() []string {
	return []string{"Show files in this folder", "Show text files to terminal", "Change folder", "Grinch password breaker", "Change user", "Opens an image"}
}

var db *sql.DB

func main() {
	var err error
	postgresUser := os.Getenv("POST_USER")
	postgresPass := os.Getenv("POST_PASS")
	postgresHost := os.Getenv("POST_HOST")
	postgresDB := os.Getenv("POST_DB")
	connStr := fmt.Sprintf("postgres://%s:%s@%s/%s", postgresUser, postgresPass, postgresHost, postgresDB)
	// Connect to database
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	ws := new(restful.WebService)

	ws.Route(ws.POST("/users").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		To(handleUser))

	ws.Route(ws.POST("/terminal").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		To(handleTerminal))

	ws.Route(ws.POST("/games").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		To(handleGames))

	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedDomains: []string{},
		AllowedMethods: []string{"POST"},
		CookiesAllowed: true,
		Container:      restful.DefaultContainer}
	restful.DefaultContainer.Filter(cors.Filter)
	restful.Add(ws)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func insertUserString() string {
	return "INSERT INTO " + userTable + "(" + userId + "," + userHash + ") VALUES ($1, $2)"
}

func insertGamesString() string {
	return "INSERT INTO " + gamesTable + "(" + userHash + ", drg, hff, left, sow) VALUES ($1, $2, $3, $4, $5)"
}

func handleTerminal(req *restful.Request, resp *restful.Response) {
	byteArr, err := io.ReadAll(req.Request.Body)
	if err != nil {
		log.Fatalf("Error on reading req body %v", req)
	}
	var termCMD TerminalCMD
	err = json.Unmarshal(byteArr, &termCMD)
	if err != nil {
		log.Printf("Could not unmarshall terminal command")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	cmds := cmdList()
	helps := cmdHelp()

	if termCMD.Command == "help" {
		var termResp TerminalResult

		for c := range cmds {
			termResp.Message += cmds[c] + " - " + helps[c] + "\n"
		}

		marshalled, err := json.Marshal(termResp)
		if err != nil {
			log.Printf("Could not marshall terminal response")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = resp.Write(marshalled)
		if err != nil {
			log.Printf("Writing user to response error %v", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	cmdParts := strings.Fields(termCMD.Command)

	switch cmdParts[0] {
	case "ls":
		filelist := files(termCMD.Location, termCMD.User)
		var termResp TerminalResult

		termResp.Message = strings.Join(filelist, "\n")

		marshalled, err := json.Marshal(termResp)
		if err != nil {
			log.Printf("Could not marshall terminal response")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = resp.Write(marshalled)
		if err != nil {
			log.Printf("Writing user to response error %v", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		break
	case "cat":
		filelist := files(termCMD.Location, termCMD.User)
		var termResp TerminalResult

		if len(cmdParts) < 2 {
			termResp.Message = "Select a file!"
		} else if slices.Contains(filelist, cmdParts[1]) {
			if cmdParts[1] == "readme.txt" {
				termResp.Message = grinchTxt
			} else if cmdParts[1] == "reminder.txt" {
				termResp.Message = santaTxt
			} else {
				termResp.Message = "Cannot display " + cmdParts[1] + " as text (but you maybe can yourself)"
			}
		} else {
			termResp.Message = "Unknown file " + cmdParts[1]
		}

		marshalled, err := json.Marshal(termResp)
		if err != nil {
			log.Printf("Could not marshall terminal response")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = resp.Write(marshalled)
		if err != nil {
			log.Printf("Writing user to response error %v", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		break
	case "cd":
		var termResp TerminalResult

		if len(cmdParts) < 2 {
			termResp.Message = "Select a folder!"
		} else if cmdParts[1] == "~" {
			termResp.NewFolder = cmdParts[1]
		} else if cmdParts[1] == "/santa_secrets/" {
			termResp.NewFolder = cmdParts[1]
		} else {
			termResp.Message = "Unknown folder " + cmdParts[1]
		}

		marshalled, err := json.Marshal(termResp)
		if err != nil {
			log.Printf("Could not marshall terminal response")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = resp.Write(marshalled)
		if err != nil {
			log.Printf("Writing user to response error %v", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		break
	case "raisins_no_choco":
		var termResp TerminalResult

		if len(cmdParts) < 2 {
			termResp.Message = "Input a guess!"
		} else if strings.ToLower(cmdParts[1]) == "rudolph" {
			termResp.Message = "Found password!\nUse `login santa 1haterudolph`"
		} else {
			if len(cmdParts) > 2 {
				termResp.Message = "Failed to crack password, too many words!"
			} else {
				termResp.Message = "Failed to crack password!"
			}
		}

		marshalled, err := json.Marshal(termResp)
		if err != nil {
			log.Printf("Could not marshall terminal response")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = resp.Write(marshalled)
		if err != nil {
			log.Printf("Writing user to response error %v", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		break
	case "login":
		var termResp TerminalResult

		if len(cmdParts) < 3 {
			termResp.Message = "Need an user and a password!"
		} else if cmdParts[1] == "santa" {
			if cmdParts[2] == "1haterudolph" {
				termResp.Message = "Successfully logged in as santa"
				termResp.NewUser = "santa"
				termResp.NewFolder = "~"
			} else {
				termResp.Message = "Wrong password!"
			}
		} else {
			termResp.Message = "Cannot login as " + cmdParts[1]
		}

		marshalled, err := json.Marshal(termResp)
		if err != nil {
			log.Printf("Could not marshall terminal response")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = resp.Write(marshalled)
		if err != nil {
			log.Printf("Writing user to response error %v", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		break
	case "open":
		filelist := files(termCMD.Location, termCMD.User)
		var termResp TerminalResult

		if len(cmdParts) < 2 {
			termResp.Message = "Select a file!"
		} else if slices.Contains(filelist, cmdParts[1]) {
			if cmdParts[1] == "invitation.jpg" {
				termResp.Redirect = "invitation.html"
			} else {
				termResp.Message = "Cannot open " + cmdParts[1] + " as image."
			}
		} else {
			termResp.Message = "Unknown file " + cmdParts[1]
		}

		marshalled, err := json.Marshal(termResp)
		if err != nil {
			log.Printf("Could not marshall terminal response")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = resp.Write(marshalled)
		if err != nil {
			log.Printf("Writing user to response error %v", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		break
	default:
		var termResp TerminalResult

		termResp.Message = fmt.Sprintf("Invalid command %v\n", cmdParts[0])

		marshalled, err := json.Marshal(termResp)
		if err != nil {
			log.Printf("Could not marshall terminal response")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = resp.Write(marshalled)
		if err != nil {
			log.Printf("Writing user to response error %v", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	return
}

func handleUser(req *restful.Request, resp *restful.Response) {
	byteArr, err := io.ReadAll(req.Request.Body)
	if err != nil {
		log.Fatalf("Error on reading req body %v", req)
	}
	var user User
	err = json.Unmarshal(byteArr, &user)
	if err != nil {
		log.Printf("Could not unmarshall user")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	h := sha512.New()
	h.Write([]byte(user.DiscordID))

	userHashed := HashedUser{UserHashID: fmt.Sprintf("%x", h.Sum(nil))}
	_, err = db.Exec(insertUserString(), user.DiscordID, userHashed.UserHashID)
	if err != nil {
		log.Printf("DB error %v", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	marshalled, err := json.Marshal(userHashed)
	if err != nil {
		log.Printf("Could not marshall user")
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = resp.Write(marshalled)
	if err != nil {
		log.Printf("Writing user to response error %v", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	return
}

func handleGames(req *restful.Request, resp *restful.Response) {
	byteArr, err := io.ReadAll(req.Request.Body)
	if err != nil {
		log.Fatalf("Error on reading req body %v", req)
	}
	var games Games
	err = json.Unmarshal(byteArr, &games)
	if err != nil {
		log.Printf("Could not unmarshall games request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = db.Exec(insertGamesString(), games.UserHashID, games.DRG, games.HFF, games.LEFT, games.SOW)
	if err != nil {
		log.Printf("DB error %v", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	return
}
