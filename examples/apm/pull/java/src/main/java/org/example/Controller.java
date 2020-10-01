package org.example;

import io.prometheus.client.*;
import io.prometheus.client.exporter.common.TextFormat;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.RequestMapping;

import java.io.IOException;
import java.io.StringWriter;
import java.io.Writer;
import java.text.DecimalFormat;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

@RestController
public class Controller {
    List<String> services = new ArrayList<String>() {{
        add("analytics");
        add("distribution");
        add("sales");
    }};

    List<String> labels = new ArrayList<String>() {{
        add("service");
        add("warning");
        add("critical");
        add("resource");
        add("group");
    }};

    String defaultResource = "FinanceServicesJava";
    String defaultGroup = "PrometheusDemo";

    @RequestMapping("/metrics")
    public String index() throws IOException {
        Writer writer = new StringWriter();
        DecimalFormat df = new DecimalFormat("#.#");
        List<Collector.MetricFamilySamples> mfs = new ArrayList<>();

        CounterMetricFamily requestsPerMinute = new CounterMetricFamily(
                "requests_per_minute",
                "Finance Services http requests per minute.",
                labels
        );

        GaugeMetricFamily bytesPerMinute = new GaugeMetricFamily(
                "bytes_per_minute",
                "Finance Services bytes transferred over http per minute.",
                labels
        );

        GaugeMetricFamily responseTime = new GaugeMetricFamily(
                "response_time",
                "Finance Services http response time average over 1 minute.",
                labels
        );

        for (String service : services) {
            // Counter multi metric building example
            requestsPerMinute.addMetric(new ArrayList<String>() {{
                add(service);                                                   // service name
                add("70");                                                      // warning threshold
                add("90");                                                      // critical threshold
                add(defaultResource);                                           // resource name
                add(defaultGroup);                                              // group name
            }}, (int) (1 + Math.random() * 100));

            // Gauge multi metric building example
            bytesPerMinute.addMetric(new ArrayList<String>() {{
                add(service);                                                   // service name
                add("40000");                                                   // warning threshold
                add("45000");                                                   // critical threshold
                add(defaultResource);                                           // resource name
                add(defaultGroup);                                              // group name
            }}, (int) (10000 + Math.random() * 50000));

            responseTime.addMetric(new ArrayList<String>() {{
                add(service);                                                   // service name
                add("2.0");                                                     // warning threshold
                add("2.5");                                                     // critical threshold
                add(defaultResource);                                           // resource name
                add(defaultGroup);                                              // group name
            }}, Double.parseDouble(df.format(Math.random() * 3)));
        }

        mfs.add(requestsPerMinute);
        mfs.add(bytesPerMinute);
        mfs.add(responseTime);

        TextFormat.write004(writer, Collections.enumeration(mfs));
        return writer.toString();
    }

    @RequestMapping("/")
    public String info() {
        return "Groundwork Prometheus Metrics example. Hit the /metrics end point to see Prometheus Exposition metrics...";
    }
}
