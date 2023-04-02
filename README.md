# artist-linker

This is a simple golang program to link Spotify artists through their collaborations.

Example usage:

`./artist-linker "rx papi" "lucki" "lil b"`

This will start at Rx Papi, and create a tree of collaborations until Lil B and LUCKI are reached.
The songs linking them are saved in a playlist on your account.

See: [Rx Papi to Lil B](https://open.spotify.com/playlist/7agox2jgRHFljiP9tcQWfL?si=17b4ab0db9d34dde),
[Rx Papi to LUCKI](https://open.spotify.com/playlist/2x0RNUaxlVbmDcwY04HqzM?si=3259a493ed594275)

Because it works this way, it's generally fastest to put the artist with the least collaborations first
to keep the tree small. A maximum depth (default 6) can be set with `-d`.

Requires setting the `SPOTIFY_ID` and `SPOTIFY_SECRET` environment variables to your Spotify developer app.
