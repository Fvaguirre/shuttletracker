import urllib.request
import json

class Stop:
    def __init__(self, id, lat, long, name):
        self.ID = id
        self.lat = lat
        self.long = long
        self.name = name
    def __str__(self):
        string = "Stop ID: "
        string += str(self.ID)
        string += ", Stop Name: "
        string += str(self.name)
        string += ", Stop latitude: "
        string += str(self.lat)
        string += ", Stop langitude: "
        string += str(self.long)
        return string
    def getID(self):
        return self.ID
    def getCoords(self):
        return (self.lat, self. long)
    def getName(self):
        return self.name
def getStops(data):
    stops = []
    for stopdict in data:
        id = stopdict["id"]
        lat = stopdict["latitude"]
        long = stopdict["longitude"]
        name = stopdict["name"]
        stops.append(Stop(id, lat, long, name))
    return stops
url = "https://shuttles.rpi.edu/stops"
response = urllib.request.urlopen(url)
data = json.loads(response.read())
stops = getStops(data)
print(stops)
# print(data)
