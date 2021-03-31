package org.groundwork.demos.prometheus;

import org.springframework.stereotype.Service;

@Service
public class DemoService {

    public DemoService(WaitProperties properties) throws InterruptedException {
        System.err.println(properties.forTime().toMillis());
        Thread.sleep(properties.forTime().toMillis());
    }

    public String go() {
        return "Hello World!";
    }
}
