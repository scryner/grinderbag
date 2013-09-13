# -*- coding:utf-8 -*-

import httplib
import datetime as dt
import time

class GrinderBag:
	def __init__(self, host, port, key, timeout=60):
		self.host = host
		self.port = port
		self.key = key
		self.timeout = timeout

	def set(self, val):
		conn = httplib.HTTPConnection(self.host, self.port)
		conn.request("POST", "/set", '{"key":"%s","val":"%s"}' % (self.key, val))
		resp = conn.getresponse()
		ret = resp.read()

		if ret == "ok":
			return True
		else:
			return False

	def clear(self):
		conn = httplib.HTTPConnection(self.host, self.port)
		conn.request("POST", "/del", '{"key":"%s"}' % self.key)
		resp = conn.getresponse()
		ret = resp.read()

		if ret == "ok":
			return True
		else:
			return False

	def sync(self):
		conn = httplib.HTTPConnection(self.host, self.port)
		now = dt.datetime.now()

		while True:
			conn.request("POST", "/get", '{"key":"%s"}' % self.key)
			resp = conn.getresponse()
			ret = resp.read()

			if ret != "None":
				return ret

			curr = dt.datetime.now()
			delta = curr - now

			if delta.seconds > self.timeout:
				break

			time.sleep(0.5)

		return None
