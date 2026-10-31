# Name My Run

A local terminal app for naming Strava runs with activity metrics at hand. It uses Go and Task.

This app was developed with the help of OpenAI Codex.

## Setup

1. Install Go (version 1.24 or newer) and [Task](https://taskfile.dev/).
2. Create your own Strava API application at [Strava API settings](https://www.strava.com/settings/api). Set its authorization callback domain to `localhost`. Strava requires a subscription to create an application.
3. In zsh, enter your application credentials without putting them in shell history:

   ```sh
   read "STRAVA_CLIENT_ID?Strava client ID: "
   read -s "STRAVA_CLIENT_SECRET?Strava client secret: "
   export STRAVA_CLIENT_ID STRAVA_CLIENT_SECRET
   ```

4. Open the following URL in a browser after replacing `YOUR_CLIENT_ID` with your client ID, and approve activity read and write access:

   ```text
   https://www.strava.com/oauth/authorize?client_id=YOUR_CLIENT_ID&redirect_uri=http%3A%2F%2Flocalhost&response_type=code&scope=activity%3Aread_all%2Cactivity%3Awrite&approval_prompt=force
   ```

   The redirect may show a connection error. Copy the `code` value from its address bar, stopping before `&`. Check the returned `scope` includes both requested permissions.

5. Exchange the code and save your credentials locally:

   ```sh
   read -s "STRAVA_AUTH_CODE?Strava authorization code: "
   export STRAVA_AUTH_CODE
   go run .
   unset STRAVA_CLIENT_ID STRAVA_CLIENT_SECRET STRAVA_AUTH_CODE
   ```

   These commands are for zsh. Credentials are stored in `.name-my-run.json` and tokens in `.strava-tokens.json`; both files are Git-ignored. Keep them private.

## Run

Run `task run`. Use `task refresh` to fetch recent activities again. Type a title and press Enter to update the activity, Ctrl+S to skip it for this session, or Escape to quit.

Activity data is cached in the Git-ignored `.strava-activities.json` file. Each person should use a separate working directory to avoid mixing accounts.

For offline approximate city lookup, run `task cities` once to download the GeoNames cities and region datasets. The app finds the nearest listed city within 25 km on your computer and caches it with the activity; coordinates are not sent to a location service. The nearest city and its state or region are approximations, not confirmed boundaries. Run `task cities` again to update the datasets. Place data is from [GeoNames](https://www.geonames.org/), licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).

When an activity has starting coordinates, the app sends its coordinates and date to [Open-Meteo](https://open-meteo.com/) to estimate weather for the start hour. Results are cached with the activity. Weather data may be unavailable or differ from the actual conditions. Open-Meteo's free service is for non-commercial use.

## Consent, privacy, and deletion

You authorize activity read and write access during setup. The app uses the Strava API to retrieve recent activities, including titles, descriptions, dates, locations, and available performance metrics; it updates a title only when you press Enter. It calculates comparisons against your cached runs. Credentials, tokens, activity details, and weather results are stored locally in the Git-ignored files listed below. Strava may monitor API usage under its policy.

To withdraw access, revoke the application under [Strava My Apps](https://www.strava.com/settings/apps). Revoking access does not automatically delete local files. To remove the local activity data and credentials, run this from the app directory:

```sh
rm -f .strava-activities.json .strava-tokens.json .name-my-run.json
```

You can verify deletion with `ls -la .strava-activities.json .strava-tokens.json .name-my-run.json`; an error for each file means it is absent. Remove any copies or backups separately. The downloaded GeoNames datasets contain public place data and are not deleted by this command.
