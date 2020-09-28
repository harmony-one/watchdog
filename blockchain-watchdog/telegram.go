package main

import (
	"github.com/yanzay/tbot/v2"
)

//func notifytg(token, ChatID, msg string) error {
func notifytg(client tbot.Client, ChatID, msg string) error {
	_, err := client.SendMessage(ChatID, msg)
	return err
}
