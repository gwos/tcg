package org.example;

import io.micrometer.core.aop.TimedAspect;
import io.micrometer.core.instrument.MeterRegistry;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.annotation.Bean;
import org.springframework.context.event.EventListener;
import org.springframework.scheduling.annotation.EnableScheduling;
import reactor.core.publisher.Flux;

import java.time.Duration;

@SpringBootApplication
@EnableScheduling
public class App {
    public static void main(String[] args) {
        SpringApplication.run(App.class, args);
    }

    private BeerService beerService;

    public App(BeerService beerService) {
        this.beerService = beerService;
    }

    @EventListener(ApplicationReadyEvent.class)
    public void orderBeers() {
        Flux.interval(Duration.ofSeconds(2))
                .map(App::toOrder)
                .doOnEach(o -> beerService.orderBeer(o.get()))
                .subscribe();
    }

    private static Order toOrder(Long l) {
        long amount = l % 5;
        String type = l % 2 == 0 ? "ale" : "light";
        return new Order((int) amount, type);
    }

    @Bean
    public TimedAspect timedAspect(MeterRegistry registry) {
        return new TimedAspect(registry);
    }
}
