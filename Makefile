all:
	@go build -o rcon.exe main.go

win:
	@GOOS=windows GOARCH=amd64 go build -o rcon.exe main.go