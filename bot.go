/* bot.go defines and builds a bot struct from use in a twitch.tv chat
 * implemented:
 *		- connects to twitch.tv
 *		- can print from console to twitch chat
 *		- concurrency via goroutines
 *		- prints to chat and console when new user joins
 *		- banning, unbanning, and timeout from chat or console (!command)
 *		- detects if a message has a website address and times out if so
 *		- exits from console with !quit
 * unimplemented:
 *		x seperate main and Bot struct into two files
 *		x react to certain phrases in chat
 *		x ability to detect spam from other users
 * by: Aaron Santucci
 * for: CS 214 Section A Spring 2017
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"strings"
	"time"
)

/* Bot: the struct for the Bot type requiring the various server
 * 		and utility information from the user
 */
type Bot struct {
	server         string
	port           string
	nickname       string
	channel        string
	automsg        string
	autoMsgCount   int
	conn           net.Conn
	mods           map[string]bool
	userLastMsg    map[string]int64
	lastMsg        int64
	maxMsgTime     int64
	userMaxLastMsg int
}

/* the constructor for the Bot struct
 * information such as nickname, channel, etc can be changed post construction
 */
func NewBot() *Bot {
	return &Bot{
		server:   "irc.twitch.tv",
		port:     "6667",
		nickname: "adefault", // the name of the bot
		channel:  "adefault", // the name of the channel
		mods:     make(map[string]bool),
		// *message stuff*
		automsg:        "This is a default test automessage",
		autoMsgCount:   5,
		conn:           nil,
		lastMsg:        0,
		maxMsgTime:     3,
		userLastMsg:    make(map[string]int64),
		userMaxLastMsg: 2,
	}
}

/* connect() connects a Bot type to the Twitch servers
 * Information for this function gained from online tutorial source:
 *		https://dinosaurscode.xyz/go/2016/08/20/tutorial-building-twitch-bot-in-go/
 */
func (bot *Bot) Connect() {
	var err error
	fmt.Printf("Connecting to Twitch server\n")
	bot.conn, err = net.Dial("tcp", bot.server+":"+bot.port)
	if err != nil {
		fmt.Printf("Cannot connect to Twitch servers")
		time.Sleep(10 * time.Second)
		bot.Connect()
	}
	fmt.Printf("Connected to " + bot.server + "\n")
}

/* Message() puts a string to Twitch chat
 * @param: message string, the string to be put in chat
 */
func (bot *Bot) Message(message string) {
	if message == "" {
		return
	}
	if bot.lastMsg+bot.maxMsgTime <= time.Now().Unix() {
		fmt.Printf("Bot: " + message + "\n")
		fmt.Fprintf(bot.conn, "PRIVMSG "+bot.channel+" :"+message+"\r\n")
		bot.lastMsg = time.Now().Unix()
	} else {
		time.Sleep(5 * time.Second)
		bot.Message(message)
	}

}

/* Automessage() starts an infinite loop that prints a Bot type's automsg variable
 *		unless it has exceeded autoMsgCount, the cooldown for messaging chat
 */
func (bot *Bot) Automessage() {
	for {
		time.Sleep(time.Duration(bot.autoMsgCount) * time.Minute)
		bot.Message(bot.automsg)
	}
}

/* ConsoleInput parses text from the commandline.
 *		To be called in an infinite (or repeating) loop until exited
 */
func (bot *Bot) ConsoleInput() {
	buffer := bufio.NewReader(os.Stdin)
	for {
		text, _ := buffer.ReadString('\n')
		if strings.HasPrefix(text, "!quit") {
			bot.Message("Shutting down bot :(")
			bot.conn.Close()
			os.Exit(0)
		} else if strings.HasPrefix(text, "!") {
			bot.ParseCommand(strings.Replace(bot.channel, "#", "", 1), text)
		} else if text != "" {
			bot.Message(text)
		}
	}
}

/* isModerator checks if a username is in a list of mods or the channel owner
 * @param: username string, the username to be checked if a moderater
 * @return: true, if username is a moderator
 *			false, if username is not a moderator
 */
func (bot *Bot) isModerator(username string) bool {
	yourChannel := strings.Replace(bot.channel, "#", "", 1)
	if bot.mods[username] == true || username == yourChannel {
		return true
	}
	return false
}

/* timeout(), ban(), and unban() "punishment" functions all receive a username
 *		and call a Twitch function from Twitch chat to act on username
 * @param: username string, the user being timed out, banned, or unbanned respectively
 */
func (bot *Bot) timeout(username string) {
	if bot.isModerator(username) {
		fmt.Printf("Unmod before punishing")
		return // exit function if user is a moderator
	} else {
		bot.Message("/timeout " + username)
		bot.Message("Timed out user: " + username)
	}
}

func (bot *Bot) ban(username string) {
	if bot.isModerator(username) {
		fmt.Printf("Unmod before punishing")
		return // exit function if user is a moderator
	} else {
		bot.Message("/ban " + username)
		bot.Message("Banned user: " + username)
	}
}

func (bot *Bot) unban(username string) {
	if bot.isModerator(username) {
		fmt.Printf("Unmod before punishing")
		return // exit function if user is a moderator
	} else {
		bot.Message("/unban " + username)
		bot.Message("Unbanned user: " + username)
	}
}

/* ParseCommand() is called by a message beginning with ":!" (if twitch chat)
 *		or "!" (if command line) and finds and call the appropriate command function
 */
func (bot *Bot) ParseCommand(user string, theCommand string) {
	if bot.isModerator(user) {
		command := strings.ToLower(theCommand)
		userinfo := strings.Split(theCommand, " ")

		if strings.HasPrefix(command, ":!timeout") || strings.HasPrefix(command, "!timeout") {
			bot.timeout(userinfo[1])
		} else if strings.HasPrefix(command, ":!ban") || strings.HasPrefix(command, "!ban") {
			bot.ban(userinfo[1])
		} else if strings.HasPrefix(command, ":!unban") || strings.HasPrefix(command, "!unban") {
			bot.unban(userinfo[1])
		}
	} else {
		bot.Message("You are not a mod " + user)
	}
}

/* main() driver instantiates a bot and gives its credentials
 * NOTE: the authentication in write "PASS" needs is refreshed about daily by Twitch;
 *		if the program isn't running it's most likely the oauth has expired
 * 		so contact me at my email for it
 */
func main() {
	channel := flag.String("channel", "ajs94", "The channel for the bot to go to")
	nickname := flag.String("nickname", "testbot", "The bot's username")
	automsg := flag.String("automessage", "This is an automessage message", "The automatic timed message")
	autoMsgCount := flag.Int("autoMsgCount", 1, "The automessage's sleep time")

	bot := NewBot()
	go bot.ConsoleInput()
	bot.Connect()

	if (*channel) != "" {
		bot.nickname = *nickname
		bot.channel = "#" + *channel
		bot.automsg = *automsg
		bot.autoMsgCount = *autoMsgCount
	}

	fmt.Printf("Giving info to server...\n")
	bot.conn.Write([]byte("PASS " + "oauth:fwnyam3yts63xngu801zmh6qhdmu9u" + "\r\n"))
	bot.conn.Write([]byte("NICK " + "testbot" + "\r\n"))
	bot.conn.Write([]byte("JOIN " + "#ajs94" + "\r\n"))
	fmt.Printf("Info given to server\n")
	fmt.Printf("Channel: " + bot.channel + "\n")

	/* the keyword "go" indicates a goroutine; a concurrent function
	 * 		in this case there are 3 concurrent funtions: Automessage, ConsoleInput, and main
	 */
	defer bot.conn.Close()
	go bot.Automessage()
	input := bufio.NewReader(bot.conn)
	tp := textproto.NewReader(input)
	go bot.ConsoleInput()

	for {
		line, err := tp.ReadLine()
		// break loop on error
		if err != nil {
			break
		}
		// split the msg
		msgParts := strings.Split(line, " ")

		// if the msg contains PING you're required to
		// respond with PONG else the bot gets kicked from twitch servers
		if msgParts[0] == "PING" {
			bot.conn.Write([]byte("PONG " + msgParts[1]))
			continue
		} else if strings.Contains(line, ".tmi.twitch.tv JOIN "+bot.channel) { // if a new user has joined
			joindata := strings.Split(line, ".tmi.twitch.tv JOIN "+bot.channel)
			userinfo := strings.Split(joindata[0], "@")
			bot.Message("PogChamp User Joined: " + userinfo[1])
		} else if strings.Contains(line, ".tmi.twitch.tv PART "+bot.channel) { // if a user has left
			joindata := strings.Split(line, ".tmi.twitch.tv PART "+bot.channel)
			userinfo := strings.Split(joindata[0], "@")
			bot.Message("BibleThump User Left: " + userinfo[1])
		} else if strings.HasPrefix(msgParts[3], ":!") { // check for commands
			userdata := strings.Split(line, ".tmi.twitch.tv PRIVMSG "+bot.channel)
			username := strings.Split(userdata[0], "@")
			usermessage := strings.Replace(userdata[1], " :", "", 1)
			fmt.Printf(username[1] + " ")
			fmt.Printf(usermessage + "\n")
			bot.ParseCommand(username[1], usermessage)
		} else if isWebsite(msgParts[3]) {
			userdata := strings.Split(line, ".tmi.twitch.tv PRIVMSG "+bot.channel)
			username := strings.Split(userdata[0], "@")
			bot.timeout(username[1])
		}
	}
}

/* isWebsite checks if a string has a website address
 * @param: theWebsite string, the message being checked
 * @return: true if theWebsite contains a url
 *			false if theWebsite does not
 */
func isWebsite(theWebsite string) bool {
	suffixes := []string{".com", ".net", ".org", ".tv", ".fm", ".gg"} // check online for more?
	// self reminder _, is the blank identifier
	for _, suffix := range suffixes {
		if strings.Contains(theWebsite, suffix) {
			return true
		}
	}
	return false
}
