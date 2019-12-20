package org.groundwork.tng.transit;

import com.sun.jna.Native;
import org.codehaus.jackson.JsonProcessingException;
import org.groundwork.rs.common.ConfiguredObjectMapper;
import org.groundwork.rs.transit.*;

import java.io.IOException;
import java.util.Collections;

public class TransitServicesImpl implements TransitServices {
    private static final String LIBTRANSIT_LIBRARY_PATH_ENV = "LIBTRANSIT";

    private ConfiguredObjectMapper objectMapper;
    private TngTransitLibrary tngTransitLibrary;
    private StringByReference errorMsg;

    public TransitServicesImpl() {
        this.objectMapper = new ConfiguredObjectMapper();
        String path = System.getenv(LIBTRANSIT_LIBRARY_PATH_ENV);
        if (path == null || path.isEmpty()) {
            path = "/home/vladislavsenkevich/Projects/groundwork/_rep/tng/gw-transit/src/main/resources/libtransit.so";
        }
        this.tngTransitLibrary = Native.load(path, TngTransitLibrary.class);
        this.errorMsg = new StringByReference(String.join("", Collections.nCopies(128, " ")));
    }

    @Override
    public boolean GoSetenv(String key, String value, StringByReference errorMsg, Integer errorMsgSize) {
        return tngTransitLibrary.GoSetenv(key, value, errorMsg, errorMsgSize);
    }

    @Override
    public void SendResourcesWithMetrics(DtoResourcesWithServices resources) throws TransitException {
        String resourcesJson = null;
        try {
            resourcesJson = objectMapper.writeValueAsString(resources);
        } catch (JsonProcessingException e) {
            throw new TransitException(e);
        } catch (IOException e) {
            e.printStackTrace();
        }

        boolean isPublished = tngTransitLibrary.SendResourcesWithMetrics(resourcesJson, errorMsg,
                errorMsg.getValue().length());
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

        boolean isPublished = tngTransitLibrary.SynchronizeInventory(inventoryJson, errorMsg,
                errorMsg.getValue().length());
        if (!isPublished) {
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public void StartNats() throws TransitException {
        if (!tngTransitLibrary.StartNats(errorMsg, errorMsg.getValue().length())) {
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public void StopNats() throws TransitException {
        tngTransitLibrary.StopNats();
    }

    @Override
    public void StartTransport() throws TransitException {
        if (!tngTransitLibrary.StartTransport(errorMsg, errorMsg.getValue().length())) {
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public void StopTransport() throws TransitException {
        if (!tngTransitLibrary.StopTransport(errorMsg, errorMsg.getValue().length())) {
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
