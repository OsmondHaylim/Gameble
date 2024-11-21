package main

import data "Gameble/data" 

func main(){
    manager := data.ClientManager{
        clients:        make(map[*data.Client]bool),
        broadcast:      make(chan []byte),
        register:       make(chan *data.Client),
        unregister:     make(chan *data.Client),
    }
}