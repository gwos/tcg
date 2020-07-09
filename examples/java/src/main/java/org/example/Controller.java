package org.example;

import io.prometheus.client.CollectorRegistry;
import io.prometheus.client.Gauge;
import io.prometheus.client.exporter.common.TextFormat;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.RequestMapping;

import java.io.IOException;
import java.io.StringWriter;
import java.io.Writer;

@RestController
public class Controller {

    public static double metricValue = 123;

    @RequestMapping("/")
    public String index() throws IOException {
        Writer writer = new StringWriter();
        CollectorRegistry registry = new CollectorRegistry();
        Gauge gauge = io.prometheus.client.Gauge.build()
                .name("java_service_example")
                .help("Description for example").register(registry);

        gauge.set(metricValue);

        TextFormat.write004(writer, registry.metricFamilySamples());
        return writer.toString();
    }
}
