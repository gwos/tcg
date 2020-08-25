package org.example;

import io.prometheus.client.CollectorRegistry;
import io.prometheus.client.Counter;
import io.prometheus.client.Gauge;
import io.prometheus.client.exporter.common.TextFormat;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.RequestMapping;

import java.io.IOException;
import java.io.StringWriter;
import java.io.Writer;

@RestController
public class Controller {
    @RequestMapping("/")
    public String index() throws IOException {
        Writer writer = new StringWriter();
        CollectorRegistry registry = new CollectorRegistry();

        Gauge gauge = io.prometheus.client.Gauge.build()
                .name("java_gauge_example")
                .help("Description for gauge")
                .labelNames("critical", "warning", "resource", "group", "unitType")
                .register(registry);
        Gauge.Child gaugeChild = gauge.labels(String.valueOf(Math.random() * 10),
                String.valueOf(Math.random() * 10), "java-server", "Prometheus-Java", "MB");
        gaugeChild.set(Math.random() * 10);

        Counter counter = io.prometheus.client.Counter.build()
                .name("java_counter_example")
                .help("Description for counter")
                .labelNames("critical", "warning", "resource", "group", "unitType")
                .register(registry);
        Counter.Child counterChild = counter.labels(String.valueOf(Math.random() * 10),
                String.valueOf(Math.random() * 10), "java-server", "Prometheus-Java", "MB");
        counterChild.inc(Math.random() * 10);


        TextFormat.write004(writer, registry.metricFamilySamples());
        return writer.toString();
    }
}
