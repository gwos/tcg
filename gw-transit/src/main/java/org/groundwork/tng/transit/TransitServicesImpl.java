package org.groundwork.tng.transit;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.sun.jna.Native;
import io.nats.streaming.Options;
import io.nats.streaming.StreamingConnection;
import io.nats.streaming.StreamingConnectionFactory;
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

    public TransitServicesImpl() {
        this.objectMapper = new ConfiguredObjectMapper();
        this.tngTransitLibrary = Native.load("/home/vladislavsenkevich/Projects/groundwork/_rep/tng/gw-transit/src/main/resources/libtransit.so", TngTransitLibrary.class);
        // TODO: load this from Maven this.tngTransitLibrary = Native.load("/Users/dtaylor/gw8/tng/libtransit/libtransit.so", TngTransitLibrary.class);
        this.errorMsg = new StringByReference("ERROR");
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

    @Override
    public boolean IsControllerRunning() throws TransitException {
        return tngTransitLibrary.IsControllerRunning();
    }

    @Override
    public boolean IsNATSRunning() throws TransitException {
        return tngTransitLibrary.IsNATSRunning();
    }

    @Override
    public boolean IsTransportRunning() throws TransitException {
        return tngTransitLibrary.IsTransportRunning();
    }

    @Override
    public void RegisterListMetricsHandler(ListMetricsCallback func) throws TransitException {
        tngTransitLibrary.RegisterListMetricsHandler(func);
    }

    @Override
    public void RemoveListMetricsHandler() throws TransitException {
        tngTransitLibrary.RemoveListMetricsHandler();
    }
}
