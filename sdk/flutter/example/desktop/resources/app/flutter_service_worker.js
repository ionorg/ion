'use strict';
const CACHE_NAME = 'flutter-app-cache';
const RESOURCES = {
  "/index.html": "e016c2926a0ee1a3bad5f19ac87d3407",
"/main.dart.js": "7ef26571d4d883a6c6cdea7dd023b6cb",
"/assets/LICENSE": "dedf3e4c3ea1c639b0c3249efbb0246f",
"/assets/AssetManifest.json": "2107f6493444b77b8d588d9962eab363",
"/assets/FontManifest.json": "23093cb5e7ab8fc344ca7d752dcca0a1",
"/assets/packages/cupertino_icons/assets/CupertinoIcons.ttf": "115e937bb829a890521f72d2e664b632",
"/assets/packages/flutter_icons/fonts/Octicons.ttf": "73b8cff012825060b308d2162f31dbb2",
"/assets/packages/flutter_icons/fonts/Feather.ttf": "6beba7e6834963f7f171d3bdd075c915",
"/assets/packages/flutter_icons/fonts/Entypo.ttf": "744ce60078c17d86006dd0edabcd59a7",
"/assets/packages/flutter_icons/fonts/MaterialCommunityIcons.ttf": "3c851d60ad5ef3f2fe43ebd263490d78",
"/assets/packages/flutter_icons/fonts/AntDesign.ttf": "3a2ba31570920eeb9b1d217cabe58315",
"/assets/packages/flutter_icons/fonts/Foundation.ttf": "e20945d7c929279ef7a6f1db184a4470",
"/assets/packages/flutter_icons/fonts/Ionicons.ttf": "b2e0fc821c6886fb3940f85a3320003e",
"/assets/packages/flutter_icons/fonts/FontAwesome.ttf": "b06871f281fee6b241d60582ae9369b9",
"/assets/packages/flutter_icons/fonts/Zocial.ttf": "5cdf883b18a5651a29a4d1ef276d2457",
"/assets/packages/flutter_icons/fonts/EvilIcons.ttf": "140c53a7643ea949007aa9a282153849",
"/assets/packages/flutter_icons/fonts/SimpleLineIcons.ttf": "d2285965fe34b05465047401b8595dd0",
"/assets/packages/flutter_icons/fonts/MaterialIcons.ttf": "a37b0c01c0baf1888ca812cc0508f6e2",
"/assets/fonts/MaterialIcons-Regular.ttf": "56d3ffdef7a25659eab6a68a3fbfaf16",
"/assets/assets/images/loading.jpeg": "4be8e22f752af2c5e151501464d3c378"
};

self.addEventListener('activate', function (event) {
  event.waitUntil(
    caches.keys().then(function (cacheName) {
      return caches.delete(cacheName);
    }).then(function (_) {
      return caches.open(CACHE_NAME);
    }).then(function (cache) {
      return cache.addAll(Object.keys(RESOURCES));
    })
  );
});

self.addEventListener('fetch', function (event) {
  event.respondWith(
    caches.match(event.request)
      .then(function (response) {
        if (response) {
          return response;
        }
        return fetch(event.request);
      })
  );
});
