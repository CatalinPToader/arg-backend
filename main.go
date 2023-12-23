package main

import (
	"crypto/sha512"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful/v3"
	_ "github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"os"
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
}

type TerminalResult struct {
	Message   string `json:"message"`
	NewFolder string `json:"new_folder"`
}

const (
	userTable = "users"
	userId    = "discord_id"
	userHash  = "hashed_id"
	grinchTxt = "Grinch here,\nGot a new idea for stopping Christmas, just hack into Santa's computer.\nTurns out instead of hashing & salting his passwords, he turns them \ninto cookies then adds milk.\n\nI uploaded a small program that can find his password, you just need to\ngive it one word from the password and it will find the rest.\nTo run it just type `raisins_no_choco <guess>`.\n\nYou should look into /SantaSecrets/ and see what you can find."
	santaTxt  = "I love my reindeers a lot,\nBut the one I love most,\nIs the one with the red nose!\n"
)

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

	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"X-My-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedDomains: []string{"*"},
		CookiesAllowed: true,
		Container:      restful.DefaultContainer}
	restful.DefaultContainer.Filter(cors.Filter)
	restful.Add(ws)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func insertUserString() string {
	return "INSERT INTO " + userTable + "(" + userId + "," + userHash + ") VALUES ($1, $2)"
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

	if termCMD.Command == "help" {
		cmds := cmdList()
		helps := cmdHelp()

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
