package org.groundwork.tng.transit;

import org.groundwork.rs.dto.DtoOperationResults;
import org.groundwork.rs.transit.DtoMetricDescriptor;
import org.groundwork.rs.transit.DtoInventory;
import org.groundwork.rs.transit.DtoResourceWithMetricsList;

import java.util.List;

public interface TransitServices {

    DtoOperationResults SendResourcesWithMetrics(DtoResourceWithMetricsList resources) throws TransitException;

    List<DtoMetricDescriptor> ListMetrics() throws TransitException;

    DtoOperationResults SynchronizeInventory(DtoInventory inventory) throws TransitException;

    void Connect(DtoCredentials credentials) throws TransitException;

    void Disconnect() throws TransitException;

    void TestNats();
}
