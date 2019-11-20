package org.groundwork.tng.transit;

import com.sun.jna.Native;
import org.codehaus.jackson.map.ObjectMapper;

import java.io.IOException;

/**
 * Hello world!
 */
public class App {
    public static void main(String[] args) throws IOException {
        TngTransitLibrary tngTransit = (TngTransitLibrary) Native.loadLibrary(
                "transitService.so", TngTransitLibrary.class);

        ObjectMapper objectMapper = new ObjectMapper();

        // Variable that will contain error message if it present in TNG
        StringByReference errorMsg = new StringByReference("ERROR");

        //TODO: Update after Kate's changes
        /** SYNCHRONIZE INVENTORY */
//        String jsonGroups = "[\n" +
//                "        {\n" +
//                "            \"groupName\": \"test-groupName1\",\n" +
//                "            \"resources\": [\n" +
//                "                {\n" +
//                "                    \"name\": \"test-name1\",\n" +
//                "                    \"type\": \"host\",\n" +
//                "                    \"description\": \"test-description1\",\n" +
//                "                    \"labels\": {\n" +
//                "                        \"k1\": \"v1\",\n" +
//                "                        \"k2\": \"v2\"\n" +
//                "                    },\n" +
//                "                    \"owner\": null\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"name\": \"test-name2\",\n" +
//                "                    \"type\": \"service\",\n" +
//                "                    \"description\": \"test-description2\",\n" +
//                "                    \"labels\": {\n" +
//                "                        \"k3\": \"v3\",\n" +
//                "                        \"k4\": \"v4\"\n" +
//                "                    },\n" +
//                "                    \"owner\": {\n" +
//                "                        \"name\": \"test-name1\",\n" +
//                "                        \"type\": \"host\",\n" +
//                "                        \"description\": \"test-description1\",\n" +
//                "                        \"labels\": {\n" +
//                "                            \"k1\": \"v1\",\n" +
//                "                            \"k2\": \"v2\"\n" +
//                "                        },\n" +
//                "                        \"owner\": null\n" +
//                "                    }\n" +
//                "                }\n" +
//                "            ]\n" +
//                "        },\n" +
//                "        {\n" +
//                "            \"groupName\": \"test-groupName2\",\n" +
//                "            \"resources\": [\n" +
//                "                {\n" +
//                "                    \"name\": \"test-name1\",\n" +
//                "                    \"type\": \"host\",\n" +
//                "                    \"description\": \"test-description1\",\n" +
//                "                    \"labels\": {\n" +
//                "                        \"k1\": \"v1\",\n" +
//                "                        \"k2\": \"v2\"\n" +
//                "                    },\n" +
//                "                    \"owner\": null\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"name\": \"test-name2\",\n" +
//                "                    \"type\": \"service\",\n" +
//                "                    \"description\": \"test-description2\",\n" +
//                "                    \"labels\": {\n" +
//                "                        \"k3\": \"v3\",\n" +
//                "                        \"k4\": \"v4\"\n" +
//                "                    },\n" +
//                "                    \"owner\": {\n" +
//                "                        \"name\": \"test-name1\",\n" +
//                "                        \"type\": \"host\",\n" +
//                "                        \"description\": \"test-description1\",\n" +
//                "                        \"labels\": {\n" +
//                "                            \"k1\": \"v1\",\n" +
//                "                            \"k2\": \"v2\"\n" +
//                "                        },\n" +
//                "                        \"owner\": null\n" +
//                "                    }\n" +
//                "                }\n" +
//                "            ]\n" +
//                "        }\n" +
//                "    ]";
//
//        String jsonInventory = "[\n" +
//                "        {\n" +
//                "            \"name\": \"test-name1\",\n" +
//                "            \"type\": \"host\",\n" +
//                "            \"description\": \"test-description1\",\n" +
//                "            \"labels\": {\n" +
//                "                \"k1\": \"v1\",\n" +
//                "                \"k2\": \"v2\"\n" +
//                "            },\n" +
//                "            \"owner\": null\n" +
//                "        },\n" +
//                "        {\n" +
//                "            \"name\": \"test-name2\",\n" +
//                "            \"type\": \"service\",\n" +
//                "            \"description\": \"test-description2\",\n" +
//                "            \"labels\": {\n" +
//                "                \"k3\": \"v3\",\n" +
//                "                \"k4\": \"v4\"\n" +
//                "            },\n" +
//                "            \"owner\": {\n" +
//                "                \"name\": \"test-name1\",\n" +
//                "                \"type\": \"host\",\n" +
//                "                \"description\": \"test-description1\",\n" +
//                "                \"labels\": {\n" +
//                "                    \"k1\": \"v1\",\n" +
//                "                    \"k2\": \"v2\"\n" +
//                "                },\n" +
//                "                \"owner\": null\n" +
//                "            }\n" +
//                "        }\n" +
//                "    ]";
//
//        String jsonResponse = tngTransit.SynchronizeInventory(jsonInventory, jsonGroups, errorMsg);

//        System.out.println(jsonResponse);


        /** LOGIN AND LOGOUT */

//        String jsonCredentials = "{\"user\":\"RESTAPIACCESS\", \"password\":\"6d2Ygwsw6dM8abSiGCaFvTyWXT8JP8XmuvwX4yynt5TH\"}";
//
//        String response = tngTransit.Connect(jsonCredentials, errorMsg);
//
//        System.out.println(response);
//
//        System.out.println(tngTransit.Disconnect(response, errorMsg));

        /** SEND RESOURCE WITH METRICS */
//
//        String response = tngTransit.SendResourcesWithMetrics(resourceWithMetricsJson, errorMsg);
//
//        System.out.println(response);
    }
}
