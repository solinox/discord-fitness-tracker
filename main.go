package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	token                string
	motivationalMessages = []string{
		"Rock on!",
		"You got this!",
		"Whoo!",
		"Whoo hoo!",
		"Those gains!",
		"Bro do you even lift? Wait I guess you do",
		"Virtual flex!",
		"",
		"Keep it up!",
		"Awesome!",
		"Party time!",
		"Amazing!",
		"Wicked Awesome!",
		"Now go get a beer.",
		"https://media1.giphy.com/media/l46CDHTqbmnGZyxKo/giphy.gif?cid=ecf05e479f9e2ff596964bc8df54c38800879a549d4668d6&rid=giphy.gif",
	}
)

const (
	dir       = "/tmp/workouts/"
	ffilename = dir + "%s.json"
	help      = "+workout (or +w for short) to add a workout\n-workout (or -w) to remove your most recent workout\n?me to view your workouts\n?all to view everyones' workouts\n?help to see this message"
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0777)
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	if strings.HasPrefix(m.Content, "+w") {
		addWorkout(m.Author.Username, m.GuildID, 1)
		n := rand.Intn(len(motivationalMessages))
		s.ChannelMessageSend(m.ChannelID, "Added. "+motivationalMessages[n])
	}

	if strings.HasPrefix(m.Content, "-w") {
		addWorkout(m.Author.Username, m.GuildID, -1)
		s.ChannelMessageSend(m.ChannelID, "Latest workout removed. Bummer")
	}

	if strings.HasPrefix(m.Content, "?me") {
		msg := queryWorkouts(m.Author.Username, m.GuildID)
		s.ChannelMessageSend(m.ChannelID, msg)
	}

	if strings.HasPrefix(m.Content, "?all") {
		msg := queryAllWorkouts(m.GuildID)
		s.ChannelMessageSend(m.ChannelID, msg)
	}

	if strings.HasPrefix(m.Content, "?help") {
		s.ChannelMessageSend(m.ChannelID, help)
	}
}

type workouts map[string][]time.Time

func addWorkout(name, guild string, count int) {
	filename := fmt.Sprintf(ffilename, guild)
	file, _ := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	data, _ := ioutil.ReadAll(file)
	var w workouts
	json.Unmarshal(data, &w)
	if w == nil {
		w = make(workouts)
	}
	if count > 0 {
		if _, ok := w[name]; !ok {
			w[name] = []time.Time{time.Now()}
		} else {
			w[name] = append(w[name], time.Now())
		}
	} else if count < 0 {
		if _, ok := w[name]; !ok {
			w[name] = []time.Time{}
		} else {
			if len(w[name]) > 0 {
				w[name] = w[name][:len(w[name])-1]
			}
		}
	}
	data, _ = json.Marshal(w)
	n, _ := file.WriteAt(data, 0)
	file.Truncate(int64(n))
	file.Sync()
	file.Close()
}

func queryWorkouts(name, guild string) string {
	filename := fmt.Sprintf(ffilename, guild)
	file, _ := os.Open(filename)
	defer file.Close()
	data, _ := ioutil.ReadAll(file)
	var w workouts
	json.Unmarshal(data, &w)
	if w == nil {
		return name + " has no workouts"
	}
	if _, ok := w[name]; !ok {
		return name + " has no workouts"
	}
	return report(name, w[name])
}

func queryAllWorkouts(guild string) string {
	filename := fmt.Sprintf(ffilename, guild)
	file, _ := os.Open(filename)
	defer file.Close()
	data, _ := ioutil.ReadAll(file)
	var w workouts
	json.Unmarshal(data, &w)
	if w == nil || len(w) == 0 {
		return "No workouts found"
	}
	ret := ""
	for k, v := range w {
		ret += report(k, v) + "\n"
	}
	return ret
}

func report(name string, workouts []time.Time) string {
	if workouts == nil || len(workouts) == 0 {
		return name + " has no workouts"
	}
	total := len(workouts)
	thisWeek := 0
	year, week := time.Now().ISOWeek()
	for i := len(workouts) - 1; i >= 0; i-- {
		y, w := workouts[i].ISOWeek()
		if y == year && w == week {
			thisWeek++
		} else {
			break // assume sorted with most recent at the end of the slice
		}
	}
	return fmt.Sprintf("%s: %d total, %d this week", name, total, thisWeek)
}
