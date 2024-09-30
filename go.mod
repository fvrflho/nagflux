module github.com/fvrflho/nagflux

go 1.14

require (
	github.com/appscode/g2 v0.0.0-20190123131438-388ba74fd273
	github.com/appscode/go v0.0.0-20201105063637-5613f3b8169f // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/golangci/golangci-lint v1.30.0
	github.com/kdar/factorlog v0.0.0-20140929220826-d5b6afb8b4fe
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/prometheus/client_golang v1.7.1
	golang.org/x/tools v0.0.0-20200820180210-c8f393745106
	gopkg.in/gcfg.v1 v1.2.3
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

replace github.com/ConSol/nagflux => github.com/fvrflho/nagflux v0.0.0-20240930214041-696511798e46

replace github.com/ConSol-Monitoring/nagflux => github.com/fvrflho/nagflux v0.0.0-20240930214041-696511798e46
