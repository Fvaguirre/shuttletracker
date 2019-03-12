import urllib.request
import datetime as dt
import json


def response(url):
    return urllib.request.urlopen(url)

def loadJSON(response):
    return json.loads(response.read())

def getRoutes():
    raiseNotDefined()

if __name__ == '__main__':
        # Currently on localhost
        url = "http://shuttles.rpi.edu/routes"

        response = urllib.request.urlopen(url)
        data = json.loads(response.read())
        print(data)
