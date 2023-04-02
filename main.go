package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/zmb3/spotify/v2"
)

const description = "created by https://github.com/egginabucket/artist-linker"

var albumTypes = []spotify.AlbumType{spotify.AlbumTypeAlbum, spotify.AlbumTypeSingle}

type (
	searcher struct {
		ctx             context.Context
		client          *spotify.Client
		startArtist     *spotify.FullArtist
		searchedArtists map[spotify.ID]bool
		finalArtists    map[spotify.ID]*spotify.FullArtist
		foundArtists    map[spotify.ID]*artistLink
		maxDepth        uint8
	}

	artistLink struct {
		prev  *artistLink
		track spotify.ID
		depth uint8
	}
)

func (al *artistLink) extend(track spotify.ID) *artistLink {
	newAl := artistLink{
		prev:  al,
		track: track,
	}
	if al != nil {
		newAl.depth = al.depth + 1
	}
	return &newAl
}

func (s *searcher) startSearch() error {
	s.searchedArtists = make(map[spotify.ID]bool)
	s.foundArtists = make(map[spotify.ID]*artistLink, len(s.finalArtists))
	start := make(map[spotify.ID]*artistLink, 0)
	start[s.startArtist.ID] = nil
	return s.search(start)
}

func (s *searcher) search(artists map[spotify.ID]*artistLink) error {
	if len(artists) == 0 {
		return fmt.Errorf("exhausted options after %d artists", len(s.searchedArtists))
	}
	newArtists := make(map[spotify.ID]*artistLink)
	newAlbums := make([]spotify.ID, 0)
	albumSources := make([]*artistLink, 0)
	for artistID, al := range artists {
		if al != nil && al.depth > s.maxDepth {
			return fmt.Errorf("depth exceeded maximum of %d", s.maxDepth)
		}
		s.searchedArtists[artistID] = true
		albums, err := s.client.GetArtistAlbums(s.ctx, artistID, albumTypes)
		if err != nil {
			return err
		}
	albums:
		for _, album := range albums.Albums {
			for _, albumArtist := range album.Artists {
				if albumArtist.ID != artistID && s.searchedArtists[albumArtist.ID] {
					continue albums
				}
			}
			newAlbums = append(newAlbums, album.ID)
			albumSources = append(albumSources, al)
		}
	}
	for i := 0; i < len(newAlbums); i += 20 {
		i2 := i + 20
		if i2 > len(newAlbums) {
			i2 = len(newAlbums)
		}
		albums, err := s.client.GetAlbums(s.ctx, newAlbums[i:i2])
		if err != nil {
			return err
		}
		for j, album := range albums {
			for _, track := range album.Tracks.Tracks {
				for _, trackArtist := range track.Artists {
					if _, exists := s.finalArtists[trackArtist.ID]; exists {
						if _, exists := s.foundArtists[trackArtist.ID]; !exists {
							s.foundArtists[trackArtist.ID] = albumSources[i+j].extend(track.ID)
							if len(s.foundArtists) == len(s.finalArtists) {
								return nil
							}
						}
					} else if !s.searchedArtists[trackArtist.ID] {
						if _, exists := newArtists[trackArtist.ID]; !exists {
							newAl := albumSources[i+j].extend(track.ID)
							fmt.Printf("Found %s at level %d\n", trackArtist.Name, newAl.depth)
							newArtists[trackArtist.ID] = newAl
						}
					}
				}
			}
		}
	}
	return s.search(newArtists)
}

func (s *searcher) savePlaylists() error {
	user, err := s.client.CurrentUser(s.ctx)
	if err != nil {
		return err
	}
	for id, al := range s.foundArtists {
		name := fmt.Sprintf("%s to %s", s.startArtist.Name, s.finalArtists[id].Name)
		pl, err := s.client.CreatePlaylistForUser(s.ctx, user.ID, name, description, false, false)
		if err != nil {
			return err
		}
		tracks := make([]spotify.ID, al.depth+1)
		for l := al; l != nil; l = l.prev {
			tracks[l.depth] = l.track
		}
		fmt.Println(tracks)
		_, err = s.client.AddTracksToPlaylist(s.ctx, pl.ID, tracks...)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var s searcher
	s.ctx = context.Background()
	maxDepth := flag.Uint("d", 6, "maximum depth")
	flag.Parse()
	s.maxDepth = uint8(*maxDepth)
	args := flag.Args()
	if len(args) < 2 {
		log.Fatalf("please give at least 2 artists to link")
	}
	http.HandleFunc("/callback", completeAuth)
	go func() {
		url := auth.AuthURL(state)
		fmt.Println("Log in to Spotify:", url)
		s.client = <-ch

		artists := make([]*spotify.FullArtist, len(args))
		for i, arg := range args {
			result, err := s.client.Search(s.ctx, arg, spotify.SearchTypeArtist)
			if err != nil {
				log.Fatal(err)
			}
			artists[i] = &result.Artists.Artists[0]
		}
		s.startArtist = artists[0]
		fmt.Println("Starting at:", s.startArtist.Name)
		s.finalArtists = make(map[spotify.ID]*spotify.FullArtist, len(artists)-1)
		for _, artist := range artists[1:] {
			s.finalArtists[artist.ID] = artist
			fmt.Println("Added destination:", artist.Name)
		}
		err := s.startSearch()
		if err != nil {
			fmt.Println(err)
		}
		err = s.savePlaylists()
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	http.ListenAndServe(":8090", nil)
}
