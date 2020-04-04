package main

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"net/http"
	"strconv"
	"time"
)
//----------------------------------------------VARIABLE----------------------------------------------------------------
var SessionCookieName = "token"
var sessionList map[string]int
//----------------------------------------------QUERY-------------------------------------------------------------------
var loginQuery =   "select ID, Username,Password,Name,Surname from tbl_user where Username = ?and [Password] = ?"
var getUserQuery = "select ID, Username,Password,Name,Surname from tbl_user where ID = ?"
var getPostQuery = "select ID, PostHeader,PostContent,PostDate,UserID from tbl_post order by PostDate desc"
var addPostQuery = "insert into tbl_post(PostHeader,PostContent,PostDate,UserID) VALUES(?,?,?,?)"

//-------------------------------------------------TEMPLATE CLASS-------------------------------------------------------
type ListTemplate struct {
	PostList     []Post
	User string
}

type LoginTemplate struct {
	Title  string
}

type Post struct {
	ID int
	PostHeader string
	PostContent  string
	PostDate string
	UserID int
	PostDateStr string
	PostUser string
}

type User struct {
	 ID int
	 Username string
	 Password string
	 Name string
	 Surname string
}


//-------------------------------------------------------DATABASE-------------------------------------------------------

func login(username string,password string) User  {
	database, _ := sql.Open("sqlite3", "db/go_post.db")
	rows, _ := database.Query(loginQuery,username,password)
	var ID int
	var Username string
	var Password string
	var Name string
	var Surname string

	for rows.Next() {
		rows.Scan(&ID, &Username, &Password,&Name,&Surname)
	}
	database.Close()
	if ID > 0{
		return User{ID,Username,Password,Name,Surname}
	}
	return User{}
}

func getUser(UserID int) User  {
	database, _ := sql.Open("sqlite3", "db/go_post.db")
	rows, _ := database.Query(getUserQuery,UserID)
	var ID int
	var Username string
	var Password string
	var Name string
	var Surname string

	for rows.Next() {
		rows.Scan(&ID, &Username, &Password,&Name,&Surname)
	}
	database.Close()
	if ID > 0{
		return User{ID,Username,Password,Name,Surname}
	}
	return User{}
}

func getPost() []Post  {
	database, _ := sql.Open("sqlite3", "db/go_post.db")
	rows, _ := database.Query(getPostQuery)
	var result = []Post{}
	for rows.Next() {
		var p Post
		_ = rows.Scan(&p.ID, &p.PostHeader, &p.PostContent, &p.PostDate, &p.UserID)
		p.PostUser = getUser(p.UserID).Username
		i, err := strconv.ParseInt(p.PostDate, 10, 64)
		if err == nil {
			tm := time.Unix(i, 0)
			p.PostDateStr = tm.Format("2006-01-02 15:04:05")
		}

		result = append(result, p)

	}
	database.Close()

	return result
}

func addPost(post Post)   {
	database, _ := sql.Open("sqlite3", "db/go_post.db")
	statement, _ := database.Prepare(addPostQuery)
	statement.Exec(post.PostHeader,post.PostContent,post.PostDate,post.UserID)
	database.Close()
}

//-------------------------------------------------HANDLER REQUEST------------------------------------------------------
func loginPost(w http.ResponseWriter,r *http.Request){
	email := r.FormValue("inputEmail")
	password := r.FormValue("inputPassword")
	loginUser := login(email,password)
	if loginUser.ID > 0 {
		guid := uuid.New().String()

		addSession(r,w,guid,loginUser.ID)

/*
		fmt.Fprintf(w, "Username = %s\n", email)
		fmt.Fprintf(w, "Password = %s\n", password)
		fmt.Fprintf(w, "Token = %s\n", guid)
*/
		postList(w,r)
	} else {

		wrongPass(w,r)
	}
}

func loginGet(w http.ResponseWriter,r *http.Request){
	clearSession(w,r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.ParseFiles("html/login.html")
	if err != nil {
		fmt.Fprintf(w, "Unable to load template")
	}

	var data = LoginTemplate{
		Title:"Sign In",
	}

	t.Execute(w, data)
}

func loginWrong(w http.ResponseWriter,r *http.Request){
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.ParseFiles("html/login.html")
	if err != nil {
		fmt.Fprintf(w, "Unable to load template")
	}

	var data = LoginTemplate{
		Title:"Wrong Username/Password",
	}

	t.Execute(w, data)
}

func postList(w http.ResponseWriter,r *http.Request){

	currentUser := checkUser(r)
	if currentUser <= 0{
		wrongPass(w,r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.ParseFiles("html/list.html")
	if err != nil {
	}
	var data = ListTemplate{
		PostList:getPost(),
		User:getUser(currentUser).Username,
	}

	t.Execute(w, data)
}

func postAddGet(w http.ResponseWriter,r *http.Request){

	currentUser := checkUser(r)
	if currentUser <= 0{
		wrongPass(w,r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, _ := template.ParseFiles("html/add.html")

	var data = ListTemplate{
		PostList:getPost(),
		User:getUser(currentUser).Username,
	}

	t.Execute(w, data)
}

func postAddPost(w http.ResponseWriter,r *http.Request){

	currentUser := checkUser(r)
	if currentUser <= 0{
		wrongPass(w,r)
		return
	}

	header := r.FormValue("inputPostHeader")
	content := r.FormValue("inputPostContent")

	time := strconv.FormatInt(time.Now().Unix(),10)
	post := Post{
		ID:          0,
		PostHeader:  header,
		PostContent: content,
		PostDate:    time,
		UserID:      currentUser,
	}
	addPost(post)
	postList(w,r)
}

func wrongPass(w http.ResponseWriter,r *http.Request){
	clearSession(w,r)
	loginWrong(w,r)
}

//-----------------------------------------------------SESSION----------------------------------------------------------

func addSession(r *http.Request,w http.ResponseWriter, guid string,ID int) {
	cookie := http.Cookie{
		Name:    SessionCookieName,
		Value:   guid,
	}
	r.AddCookie(&cookie)
	sessionList[guid] = ID
	http.SetCookie(w, &cookie)

}

func clearSession(w http.ResponseWriter,r *http.Request) {
	cookie := &http.Cookie{
		Name:   SessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
	token := getToken(r)
	sessionList[token] = 0
}

func checkUser(r *http.Request) int {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return 0
	}
	userID := 0
	userID , _ = sessionList[cookie.Value]
	return userID
}

func getToken(r *http.Request) string{
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}


func main() {
	sessionList = make(map[string]int)
	http.HandleFunc("/login", loginGet)
	http.HandleFunc("/loginPost", loginPost)
	http.HandleFunc("/logout", loginGet)
	http.HandleFunc("/postList", postList)
	http.HandleFunc("/postAdd", postAddGet)
	http.HandleFunc("/postAddPost", postAddPost)

	fmt.Printf("Server is running.. test -->> http://localhost:8080/login\n\r")

	http.ListenAndServe(":8080", nil)
}
