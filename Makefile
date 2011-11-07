include $(GOROOT)/src/Make.inc

TARG = streamfeed
GOFILES = \
	usgs.go \
    main.go \

include $(GOROOT)/src/Make.cmd
