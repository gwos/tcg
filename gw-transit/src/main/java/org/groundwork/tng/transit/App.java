package org.groundwork.tng.transit;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.jna.Native;

import java.io.IOException;

/**
 * Hello world!
 *
 */
public class App 
{
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

//        String jsonCredentials = "{\"user\":\"RESTAPIACCESS\", \"password\":\"***REMOVED***\"}";
//
//        String response = tngTransit.Connect(jsonCredentials, errorMsg);
//
//        System.out.println(response);
//
//        System.out.println(tngTransit.Disconnect(response, errorMsg));

        /** SEND RESOURCE WITH METRICS */

//        String resourceWithMetricsJson = "[\n" +
//                "        {\n" +
//                "            \"resource\":{\n" +
//                "                \"name\":\"mc-test-host\",\n" +
//                "                \"type\":\"HOST\",\n" +
//                "                \"status\":\"HOST_UP\"\n" +
//                "            }\n" +
//                "        },\n" +
//                "        {\n" +
//                "            \"resource\":{\n" +
//                "                \"name\":\"mc-test-service-0\",\n" +
//                "                \"type\":\"SERVICE\",\n" +
//                "                \"status\":\"SERVICE_OK\",\n" +
//                "                \"owner\":\"mc-test-host\"\n" +
//                "            },\n" +
//                "            \"metrics\":[\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0\",\n" +
//                "                    \"sampleType\":\"Value\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.805+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"StringType\",\n" +
//                "                        \"stringValue\":\"5.5\"\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0\",\n" +
//                "                    \"sampleType\":\"Warning\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.805+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"StringType\",\n" +
//                "                        \"stringValue\":\"7\"\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0\",\n" +
//                "                    \"sampleType\":\"Critical\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.805+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"StringType\",\n" +
//                "                        \"stringValue\":\"10\"\n" +
//                "                    }\n" +
//                "                }\n" +
//                "            ]\n" +
//                "        },\n" +
//                "        {\n" +
//                "            \"resource\":{\n" +
//                "                \"name\":\"mc-test-service-1\",\n" +
//                "                \"type\":\"SERVICE\",\n" +
//                "                \"status\":\"SERVICE_OK\",\n" +
//                "                \"owner\":\"mc-test-host\"\n" +
//                "            },\n" +
//                "            \"metrics\":[\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-A\",\n" +
//                "                    \"sampleType\":\"Value\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.705+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"DoubleType\",\n" +
//                "                        \"doubleValue\":5.5\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-A\",\n" +
//                "                    \"sampleType\":\"Warning\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.705+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"IntegerType\",\n" +
//                "                        \"integerValue\":7\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-A\",\n" +
//                "                    \"sampleType\":\"Critical\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.705+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"IntegerType\",\n" +
//                "                        \"integerValue\":10\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-A\",\n" +
//                "                    \"sampleType\":\"Min\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.705+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"IntegerType\",\n" +
//                "                        \"integerValue\":0\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-A\",\n" +
//                "                    \"sampleType\":\"Max\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.705+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"IntegerType\",\n" +
//                "                        \"integerValue\":15\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                        \"cpu\":\"cpu0\"\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-B\",\n" +
//                "                    \"sampleType\":\"Value\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.805+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.905+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"DoubleType\",\n" +
//                "                        \"doubleValue\":1.0\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                        \"cpu\":\"cpu1\"\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-B\",\n" +
//                "                    \"sampleType\":\"Value\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.805+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.905+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"DoubleType\",\n" +
//                "                        \"doubleValue\":1.1\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                        \"cpu\":\"cpu2\"\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-B\",\n" +
//                "                    \"sampleType\":\"Value\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.805+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.905+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"DoubleType\",\n" +
//                "                        \"doubleValue\":0.9\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                        \"x\":\"x0\"\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-C\",\n" +
//                "                    \"sampleType\":\"Value\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.805+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"StringType\",\n" +
//                "                        \"stringValue\":\"10\"\n" +
//                "                    }\n" +
//                "                },\n" +
//                "                {\n" +
//                "                    \"tags\":{\n" +
//                "                        \"x\":\"x1\"\n" +
//                "                    },\n" +
//                "                    \"metricName\":\"mc-test-service-0-C\",\n" +
//                "                    \"sampleType\":\"Value\",\n" +
//                "                    \"interval\":{\n" +
//                "                        \"startTime\":\"2019-09-27T06:55:56.805+0000\",\n" +
//                "                        \"endTime\":\"2019-09-27T06:55:56.805+0000\"\n" +
//                "                    },\n" +
//                "                    \"value\":{\n" +
//                "                        \"valueType\":\"StringType\",\n" +
//                "                        \"stringValue\":\"12\"\n" +
//                "                    }\n" +
//                "                }\n" +
//                "            ]\n" +
//                "        }\n" +
//                "    ]";
//
//        String response = tngTransit.SendResourcesWithMetrics(resourceWithMetricsJson, errorMsg);
//
//        System.out.println(response);
    }
}
