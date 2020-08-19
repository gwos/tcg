import java.io.BufferedReader;
import java.io.DataOutputStream;
import java.io.IOException;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;


public class GWClient {
    private static final String LOGIN_URL = "/api/auth/login";
    private static final String LOGOUT_URL = "/api/auth/logout";

    private String host;
    private String user;
    private String password;
    private String gwosAppName;
    private String gwosApiToken;

    public GWClient(String host, String user, String password, String gwosAppName) {
        this.host = host;
        this.user = user;
        this.password = password;
        this.gwosAppName = gwosAppName;
    }

    public String getHost() {
        return host;
    }

    public void setHost(String host) {
        this.host = host;
    }

    public String getUser() {
        return user;
    }

    public void setUser(String user) {
        this.user = user;
    }

    public String getGwosAppName() {
        return gwosAppName;
    }

    public void setGwosAppName(String gwosAppName) {
        this.gwosAppName = gwosAppName;
    }

    public String getGwosApiToken() {
        return gwosApiToken;
    }

    public void setGwosApiToken(String gwosApiToken) {
        this.gwosApiToken = gwosApiToken;
    }

    public void login() throws IOException {
        String urlParameters = "user=" + this.user + "&password=" + this.password
                + "&gwos-app-name=" + this.gwosAppName;

        byte[] postData = urlParameters.getBytes(StandardCharsets.UTF_8);
        int postDataLength = postData.length;

        URL obj = new URL(host + LOGIN_URL);
        HttpURLConnection con = (HttpURLConnection) obj.openConnection();
        con.setRequestMethod("POST");
        con.setInstanceFollowRedirects(false);
        con.setRequestProperty("Content-Type", "application/x-www-form-urlencoded");
        con.setRequestProperty("charset", "utf-8");
        con.setRequestProperty("Content-Length", Integer.toString(postDataLength));
        con.setUseCaches(false);
        con.setDoOutput(true);
        try (DataOutputStream wr = new DataOutputStream(con.getOutputStream())) {
            wr.write(postData);
        }

        try (BufferedReader br = new BufferedReader(
                new InputStreamReader(con.getInputStream(), StandardCharsets.UTF_8))) {
            StringBuilder response = new StringBuilder();
            String responseLine;
            while ((responseLine = br.readLine()) != null) {
                response.append(responseLine.trim());
            }
            this.gwosApiToken = response.toString();
        }
    }

    public void logout() throws IOException {
        String urlParameters = "gwos-app-name=" + this.gwosAppName
                + "&gwos-api-token=" + this.gwosApiToken;

        byte[] postData = urlParameters.getBytes(StandardCharsets.UTF_8);
        int postDataLength = postData.length;

        URL obj = new URL(host + LOGOUT_URL);
        HttpURLConnection con = (HttpURLConnection) obj.openConnection();
        con.setRequestMethod("POST");
        con.setInstanceFollowRedirects(false);
        con.setRequestProperty("Content-Type", "application/x-www-form-urlencoded");
        con.setRequestProperty("GWOS-APP-NAME", this.gwosAppName);
        con.setRequestProperty("GWOS-API-TOKEN", this.gwosApiToken);
        con.setRequestProperty("charset", "utf-8");
        con.setRequestProperty("Content-Length", Integer.toString(postDataLength));
        con.setUseCaches(false);
        con.setDoOutput(true);
        try (DataOutputStream wr = new DataOutputStream(con.getOutputStream())) {
            wr.write(postData);
        }

        try (BufferedReader br = new BufferedReader(
                new InputStreamReader(con.getInputStream(), StandardCharsets.UTF_8))) {
            StringBuilder response = new StringBuilder();
            String responseLine;
            while ((responseLine = br.readLine()) != null) {
                response.append(responseLine.trim());
            }
            System.out.println("Logout: " + response.toString());
        }
    }
}
