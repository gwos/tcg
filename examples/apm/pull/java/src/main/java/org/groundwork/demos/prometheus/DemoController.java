package org.groundwork.demos.prometheus;

import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import javax.servlet.http.HttpServletResponse;
import java.util.Map;
import java.util.Random;

@RestController
public class DemoController {

    private DemoService demoService;

    private String SIMPLE_PROMETHEUS_METRICS =
       """
            # HELP simple_calculated using calculated monitor status
            # TYPE simple_calculated counter
            simple_calculated{service="simple-service-1",warning="80",critical="90",resource="AppGenerated",group="SpringBoot"} %d
            # HELP simple_metric and passing in the status code
            # TYPE simple_metric counter
            simple_metric{service="simple-service-1",resource="AppGenerated",status="%s"} %d
        """;

    private Map<Integer, String> MONITOR_STATUS = Map.of(0, "OK", 1, "WARNING", 2, "CRITICAL");

    public DemoController(DemoService demoService) {
        this.demoService = demoService;
    }

    @RequestMapping("/hello")
    public String hello() {
        return demoService.go();
    }

    @RequestMapping("/simple")
    public String prometheusSimpleEndPoint(HttpServletResponse response) {
        Random random = new Random();
        response.setStatus(random.nextInt(2) > 0 ? 220 : 200);
        return String.format(SIMPLE_PROMETHEUS_METRICS, random.nextInt(100), MONITOR_STATUS.get(random.nextInt(3)), random.nextInt(100));
    }
}
