package org.groundwork.tng.transit;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.jna.Native;
import org.groundwork.rs.transit.*;

import java.io.IOException;
import java.util.List;

public class TransitServicesImpl implements TransitServices {

    private ObjectMapper objectMapper;
    private TngTransitLibrary tngTransitLibrary;
    private StringByReference errorMsg;

    public TransitServicesImpl() {
        this.objectMapper = new ObjectMapper();
        this.objectMapper.setSerializationInclusion(JsonInclude.Include.NON_NULL);
        this.tngTransitLibrary = Native.load("/home/vsenkevich/Projects/effectivesoft/groundwork/_rep/tng/gw-transit/src/main/resources/libtransit.so", TngTransitLibrary.class);
        // TODO: load this from Maven this.tngTransitLibrary = Native.load("/Users/dtaylor/gw8/tng/libtransit/libtransit.so", TngTransitLibrary.class);
        this.errorMsg = new StringByReference("ERROR");
    }

    @Override
    public void SendResourcesWithMetrics(DtoResourceWithMetricsList resources) throws TransitException {
        String resourcesJson;
        try {
            resourcesJson = objectMapper.writeValueAsString(resources);
        } catch (JsonProcessingException e) {
            throw new TransitException(e);
        }

        boolean isPublished = tngTransitLibrary.SendResourcesWithMetrics(resourcesJson, errorMsg);
        if (!isPublished) {
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public List<DtoMetricDescriptor> ListMetrics() throws TransitException {
        String metricDescriptorListJson = tngTransitLibrary.ListMetrics(errorMsg);
        if (metricDescriptorListJson == null) {
            throw new TransitException(errorMsg.getValue());
        }

        try {
            return objectMapper.readValue(metricDescriptorListJson, new TypeReference<List<DtoMetricDescriptor>>() {
            });
        } catch (IOException e) {
            throw new TransitException(e);
        }
    }

    @Override
    public void SynchronizeInventory(DtoInventory inventory) throws TransitException {
        String inventoryJson;
        try {
            inventoryJson = objectMapper.writeValueAsString(inventory);
        } catch (JsonProcessingException e) {
            throw new TransitException(e);
        }

        boolean isPublished = tngTransitLibrary.SynchronizeInventory(inventoryJson, errorMsg);
        if (!isPublished) {
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public void StartNATS() throws TransitException {
        if(!tngTransitLibrary.StartNATS(errorMsg)){
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public void StopNATS() throws TransitException {
        tngTransitLibrary.StopNATS();
    }

    @Override
    public void StartTransport() throws TransitException {
        if(!tngTransitLibrary.StartTransport(errorMsg)){
            throw new TransitException(errorMsg.getValue());
        }
    }

    @Override
    public void StopTransport() throws TransitException {
        if(!tngTransitLibrary.StopTransport(errorMsg)){
            throw new TransitException(errorMsg.getValue());
        }
    }
}