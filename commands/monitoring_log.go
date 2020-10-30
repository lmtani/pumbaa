package commands

import (
	"github.com/lmtani/cromwell-cli/pkg/input"
	"github.com/urfave/cli/v2"
)

func Monitoring(c *cli.Context) error {
	err := input.ReadFile("bioinfo-dev-temp", "NanoporeAlignment/c095dfcc-f684-4bb2-baa8-bba29bcee186/call-GetFastqFiles/stderr")
	if err != nil {
		return err
	}
	// cpu := []float64{
	// 	5.03145,
	// 	12.4069,
	// 	51.341,
	// 	9.20555,
	// 	7.77917,
	// 	6.95322,
	// 	5.77889,
	// 	2.01511,
	// 	9.92462,
	// 	7.4401,
	// 	10.3448,
	// 	34.0233,
	// 	15.4337,
	// 	3.25,
	// 	7.86802,
	// 	19.0537,
	// 	18.8462,
	// 	2.91139,
	// 	8.88325,
	// 	10.6599,
	// 	10.5128,
	// 	9.21717,
	// 	14.2313,
	// 	4.30925,
	// 	5.70342,
	// 	12.3737,
	// 	5.30303,
	// 	21.4195,
	// 	21.9388,
	// 	6.4557,
	// 	2.63488,
	// 	2.37797,
	// 	20.0501,
	// 	2.37797,
	// 	4.00501,
	// 	17.5505,
	// 	1.25628,
	// 	9.31677,
	// 	7.59494,
	// 	7.04403,
	// 	2.88582,
	// 	2.49688,
	// 	14.3216,
	// 	22.896,
	// 	10.3535,
	// 	7.25907,
	// 	9,
	// 	7.09759,
	// 	13.9417,
	// 	3.50438,
	// 	15.1399,
	// 	2.62829,
	// 	10.5395,
	// 	6.17907,
	// 	25.6927,
	// 	4.8995,
	// 	4.40806,
	// 	3.65239,
	// 	9.67337,
	// 	5.02513,
	// }

	// color.Cyan("CPU usage (%)")
	// option := asciigraph.Height(10)
	// graph := asciigraph.Plot(cpu, option)
	// fmt.Println(graph)

	return nil
}
