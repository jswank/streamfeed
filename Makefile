include $(GOROOT)/src/Make.inc

TARG = streamfeed
GOFILES = \
	usgs.go \
	config.go \
	main.go \

include $(GOROOT)/src/Make.cmd
