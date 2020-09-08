package org.example;

import io.prometheus.client.*;
import io.prometheus.client.exporter.common.TextFormat;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.RequestMapping;

import java.io.IOException;
import java.io.StringWriter;
import java.io.Writer;
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

    List<String> nodes = new ArrayList<String>() {{
        add("node_1");
        add("node_2");
    }};

    List<String> labels = new ArrayList<String>() {{
        add("node");
        add("service");
        add("code");
        add("warning");
        add("critical");
        add("resource");
        add("group");
    }};

    String defaultResource = "FinanceServicesJava";
    String defaultGroup = "PrometheusDemo";

    @RequestMapping("/")
    public String index() throws IOException {
        Writer writer = new StringWriter();
        List<Collector.MetricFamilySamples> mfs = new ArrayList<>();

        CounterMetricFamily counterFamily = new CounterMetricFamily(
                "requests_total",
                "Finance Services total http requests made.",
                labels
        );

        GaugeMetricFamily gaugeFamily = new GaugeMetricFamily(
                "bytes_transferred",
                "Finance Services total http requests made.",
                labels
        );

        for (String service : services) {
            for (String node : nodes) {
                // Counter multi metric building example
                counterFamily.addMetric(new ArrayList<String>() {{
                    add(node);                                              // node value
                    add(service);                                           // service name
                    add("200");                                             // code value
                    add(String.valueOf((int) (1 + Math.random() * 10)));    // warning threshold
                    add(String.valueOf((int) (1 + Math.random() * 10)));    // critical threshold
                    add(defaultResource);                                   // resource name
                    add(defaultGroup);                                      // group name
                }}, (int) (1 + Math.random() * 10));

                // Gauge multi metric building example
                gaugeFamily.addMetric(new ArrayList<String>() {{
                    add(node);                                              // node value
                    add(service);                                           // service name
                    add("500");                                             // code value
                    add(String.valueOf((int) (1 + Math.random() * 10)));    // warning threshold
                    add(String.valueOf((int) (1 + Math.random() * 10)));    // critical threshold
                    add(defaultResource);                                   // resource name
                    add(defaultGroup);                                      // group name
                }}, (int) (1 + Math.random() * 10));
            }
        }

        mfs.add(counterFamily);
        mfs.add(gaugeFamily);

        TextFormat.write004(writer, Collections.enumeration(mfs));
        return writer.toString();
    }
}
