import io.prometheus.client.exporter.HttpConnectionFactory;

import java.io.IOException;
import java.net.HttpURLConnection;
import java.net.URL;

public class CustomHttpConnectionFactory implements HttpConnectionFactory {
    public static String GWOS_APP_NAME = "GW8";
    public static String GWOS_API_TOKEN = "7be44510-c61b-4fb1-bbad-94bd6806c454";

    public HttpURLConnection create(String url) throws IOException {
        URL obj = new URL(url);
        HttpURLConnection con = (HttpURLConnection) obj.openConnection();
        con.setRequestProperty("GWOS-APP-NAME", GWOS_APP_NAME);
        con.setRequestProperty("GWOS-API-TOKEN", GWOS_API_TOKEN);
        return con;
    }
}
