module github.com/harmony-one/watchdog

go 1.14

require (
	github.com/PagerDuty/go-pagerduty v0.0.0-20190829185950-7180e89b583b
	github.com/Workiva/go-datastructures v1.0.50
	github.com/ahmetb/go-linq v3.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/harmony-one/watchdog/internal/box v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	github.com/takama/daemon v0.11.0
	github.com/valyala/fasthttp v1.2.0
	gopkg.in/yaml.v2 v2.2.7
)

replace github.com/harmony-one/watchdog/internal/box => ./internal/box
