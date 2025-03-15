build:
	go mod tidy
	go build
clean:
	rm -f *.o resp-cost