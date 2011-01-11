include $(GOROOT)/src/Make.inc

TARG=gossip
GOFMT=gofmt

GOFILES=\
	udp.go\

include $(GOROOT)/src/Make.pkg

format:
	${GOFMT} -w -s udp.go
	${GOFMT} -w -s udp_test.go
