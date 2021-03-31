package org.groundwork.demos.prometheus;

import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.boot.context.properties.ConstructorBinding;
import org.springframework.boot.context.properties.bind.DefaultValue;
    import org.springframework.boot.context.properties.bind.Name;
import org.springframework.boot.convert.DurationUnit;

import java.time.Duration;
import java.time.temporal.ChronoUnit;

@ConfigurationProperties("wait")
@ConstructorBinding
public class  WaitProperties {

    private Duration forTime;

    public WaitProperties(@DefaultValue("0") @DurationUnit(ChronoUnit.SECONDS) @Name("for") Duration forTime) {
        this.forTime = forTime;
    }

    public Duration forTime() {
        return forTime;
    }
}
