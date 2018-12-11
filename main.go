package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "strings"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    addr   = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
    url    = flag.String("url", "", "The url to monitor.")
    prefix = flag.String("prefix", "json", "The prefix for the prometheus keys.")
    up     = flag.Bool("up", true, "Add up node to metrics.")
)

var (
    monitors = make(map[string]prometheus.Gauge)
)

func init() {

}

func translate_prometheus_subkey(key string) string {
    return strings.Replace(strings.Replace(key, ".", "_", -1), "-", "_", -1)
}

func translate_prometheus_key(key string) string {
    return *prefix + "_" + translate_prometheus_subkey(key)
}

func do_fetch_data() map[string]float64 {
    resp, err := http.Get(*url)
    if err != nil {
        fmt.Println(err)
        if !*up {
            return nil
        } else {
            data := make(map[string]float64)
            data["up"] = 0
            return data
        }
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println(err)
        if !*up {
            return nil
        } else {
            data := make(map[string]float64)
            data["up"] = 0
            return data
        }
    }

    data := make(map[string]float64)
    err = json.Unmarshal(body, &data)
    if err != nil {
        fmt.Println(err)
        if !*up {
            return nil
        } else {
            data["up"] = 0
            return data
        }
    }

    if *up {
        data["up"] = 1
    }
    return data
}

func update_data() {
    data := do_fetch_data()
    for k, v := range data {
        k = translate_prometheus_key(k)
        monitor := monitors[k]
        monitor.(prometheus.Gauge).Set(v)
    }
}

func prometheus_timer_setting() {
    timer := time.NewTicker(1 * time.Second)
    for {
        select {
        case <-timer.C:
            update_data()
        }
    }
}

func init_prometheus() {
    data := do_fetch_data()
    for k, _ := range data {
        mainkey := translate_prometheus_key(k)
        k = translate_prometheus_subkey(k)
        var gauge = prometheus.NewGauge(prometheus.GaugeOpts{
            Name: prometheus.BuildFQName(*prefix, "", k),
            Help: k,
        })
        monitors[mainkey] = gauge
        prometheus.MustRegister(gauge)
    }
}

func main() {
    flag.Parse()

    init_prometheus()
    go prometheus_timer_setting()

    // Expose the registered metrics via HTTP.
    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(*addr, nil))
}
