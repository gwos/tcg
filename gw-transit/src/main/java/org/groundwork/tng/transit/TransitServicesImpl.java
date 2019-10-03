package org.groundwork.tng.transit;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.jna.Native;
import org.groundwork.rs.dto.DtoOperationResults;
import org.groundwork.rs.transit.*;

import java.io.IOException;
import java.util.List;

public class TransitServicesImpl implements TransitServices {

    private ObjectMapper objectMapper;
    private TngTransitLibrary tngTransitLibrary;
    private StringByReference errorMsg;

    public TransitServicesImpl() {
        this.objectMapper = new ObjectMapper();
        this.tngTransitLibrary = Native.loadLibrary("libtransit.so", TngTransitLibrary.class);
        this.errorMsg = new StringByReference("ERROR");
    }

    @Override
    public DtoOperationResults SendResourcesWithMetrics(DtoResourceWithMetricsList resources) throws TransitException {
        String resourcesJson;
        try {
            resourcesJson = objectMapper.writeValueAsString(resources);
        } catch (JsonProcessingException e) {
            throw new TransitException(e);
        }

        String operationResultsJson = tngTransitLibrary.SendResourcesWithMetrics(resourcesJson, errorMsg);
        if (operationResultsJson == null) {
            throw new TransitException(errorMsg.getValue());
        }

        try {
            return objectMapper.readValue(operationResultsJson, DtoOperationResults.class);
        } catch (IOException e) {
            throw new TransitException(e);
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
    public DtoOperationResults SynchronizeInventory(DtoInventory inventory) throws TransitException {
        String inventoryJson;
        try {
            inventoryJson = objectMapper.writeValueAsString(inventory);
        } catch (JsonProcessingException e) {
            throw new TransitException(e);
        }

        String operationResultsJson = tngTransitLibrary.SynchronizeInventory(inventoryJson, errorMsg);
        if (operationResultsJson == null) {
            throw new TransitException(errorMsg.getValue());
        }

        try {
            return objectMapper.readValue(operationResultsJson, DtoOperationResults.class);
        } catch (IOException e) {
            throw new TransitException(e);
        }
    }
}



