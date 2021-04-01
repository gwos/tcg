package org.groundwork.demos.prometheus;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;

@SpringBootApplication
@ConfigurationPropertiesScan
public class PrometheusDemoApplication {

	public static void main(String[] args) {
		// SpringApplication.run(PrometheusDemoApplication.class, args);
		SpringApplication app = new SpringApplication(PrometheusDemoApplication.class);
		// get metrics out of spring itself, allows to get much more detailed heuristics about app
		// plug in Java Flight Recorder into Spring
		// app.setApplicationStartup(new FlightRecorderApplicationStartup());
		// app.setApplicationStartup(new BufferingApplicationStartup(10_000));
		app.run(args);
	}

}
