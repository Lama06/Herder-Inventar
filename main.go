package main

func main() {
	s, err := newServer()
	if err != nil {
		panic(err)
	}
	err = s.start()
	if err != nil {
		panic(err)
	}
}
