import io.prometheus.client.CollectorRegistry;
import io.prometheus.client.Counter;
import io.prometheus.client.Gauge;
import io.prometheus.client.exporter.PushGateway;
import org.apache.commons.math3.util.Precision;

public class Main {

    public static void main(String[] args) throws Exception {
        executeBatchJob();
    }

    static void executeBatchJob() throws Exception {
        CollectorRegistry registry = new CollectorRegistry();
        Gauge duration = Gauge.build()
                .name("java_gauge_example")
                .labelNames("critical", "warning", "resource", "group", "unitType")
                .help("Gauge example in Java")
                .register(registry);
        Gauge.Child durationChild = duration.labels(String.valueOf(Math.random() * 40), String.valueOf(Math.random() * 40),
                "Prometheus-Java-Push", "Prometheus-Java", "MB");
        durationChild.set(Precision.round(Math.random() * 40, 3));

        Counter counter = Counter.build()
                .name("java_counter_example")
                .help("Counter example in Java")
                .labelNames("critical", "warning", "resource", "group", "unitType")
                .register(registry);
        Counter.Child counterChild = counter.labels(String.valueOf(Math.random() * 40), String.valueOf(Math.random() * 40),
                "Prometheus-Java-Push", "Prometheus-Java", "MB");
        counterChild.inc(Precision.round(Math.random() * 40, 3));

        PushGateway pg = new PushGateway("localhost:8099/api/v1");
        pg.setConnectionFactory(new CustomHttpConnectionFactory());
        pg.pushAdd(registry, "my_batch_job");
    }
}
