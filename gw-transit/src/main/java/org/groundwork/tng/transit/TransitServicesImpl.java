package org.groundwork.tng.transit;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.sun.jna.Native;
import io.nats.streaming.Options;
import io.nats.streaming.StreamingConnection;
import io.nats.streaming.StreamingConnectionFactory;
import io.nats.streaming.Subscription;
import org.groundwork.rs.common.ConfiguredObjectMapper;
import org.groundwork.rs.transit.*;

import java.io.IOException;
import java.util.concurrent.TimeoutException;

public class TransitServicesImpl implements TransitServices {
    private static final String TNG_NATS_URL = "nats://localhost:4222";
    private static final String CLIENT_ID = "gw-transit";
    private static final String CLUSTER_ID = "tng-cluster";
    private static final Integer WAIT_FOR_NATS_SERVER = 1000;

    private ConfiguredObjectMapper objectMapper;
    private TngTransitLibrary tngTransitLibrary;
    private StringByReference errorMsg;

    public TransitServicesImpl() throws IOException, InterruptedException, TimeoutException {
        this.objectMapper = new ConfiguredObjectMapper();
        this.tngTransitLibrary = Native.load("/home/vladislavsenkevich/Projects/groundwork/_rep/tng/gw-transit/src/main/resources/libtransit.so", TngTransitLibrary.class);
        // TODO: load this from Maven this.tngTransitLibrary = Native.load("/Users/dtaylor/gw8/tng/libtransit/libtransit.so", TngTransitLibrary.class);
        this.errorMsg = new StringByReference("ERROR");

        Thread.sleep(WAIT_FOR_NATS_SERVER);

        StreamingConnectionFactory cf = new StreamingConnectionFactory(new Options.Builder()
                .natsUrl(TNG_NATS_URL)
                .clusterId(CLUSTER_ID)
                .clientId(CLIENT_ID)
                .build());

        StreamingConnection sc = cf.createConnection();
        sc.subscribe("list-metrics-request", m -> {
            try {
                sc.publish("list-metrics-response", this.ListMetrics());
            } catch (IOException | InterruptedException | TimeoutException e) {
                e.printStackTrace();
            }
        });
    }

    @Override
    public void SendResourcesWithMetrics(DtoResourceWithMetricsList resources) throws TransitException {
        String resourcesJson = null;
        try {
            resourcesJson = objectMapper.writeValueAsString(resources);
        } catch (JsonProcessingException e) {
            throw new TransitException(e);
        } catch (IOException e) {
            e.printStackTrace();
        }

        boolean isPublished = tngTransitLibrary.SendResourcesWithMetrics(resourcesJson, errorMsg);
        if (!isPublished) {
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public byte[] ListMetrics() throws TransitException {
        return "RESPONSE".getBytes();
    }

    @Override
    public void SynchronizeInventory(DtoInventory inventory) throws TransitException {
        String inventoryJson = null;
        try {
            inventoryJson = objectMapper.writeValueAsString(inventory);
        } catch (JsonProcessingException e) {
            throw new TransitException(e);
        } catch (IOException e) {
            e.printStackTrace();
        }

        boolean isPublished = tngTransitLibrary.SynchronizeInventory(inventoryJson, errorMsg);
        if (!isPublished) {
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public void StartNATS() throws TransitException {
        if (!tngTransitLibrary.StartNATS(errorMsg)) {
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public void StopNATS() throws TransitException {
        tngTransitLibrary.StopNATS();
    }

    @Override
    public void StartTransport() throws TransitException {
        if (!tngTransitLibrary.StartTransport(errorMsg)) {
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public void StopTransport() throws TransitException {
        if (!tngTransitLibrary.StopTransport(errorMsg)) {
            throw new TransitException(errorMsg.getValue());
        }
    }
}
