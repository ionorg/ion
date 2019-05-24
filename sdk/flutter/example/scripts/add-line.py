#!/usr/bin/python
# -*- coding: UTF-8 -*-

import sys, getopt

def insertNewLine(file_name, line_number, text):
  with open(file_name, 'r+') as fh:
    lines = fh.readlines()
    fh.seek(0)
    lines.insert(line_number - 1, text + '\n')
    fh.writelines(lines)

def main(argv):
   inputfile = ''
   line = ''
   text = ''
   try:
      opts, args = getopt.getopt(argv,"hi:n:t:")
   except getopt.GetoptError:
      print 'add-line.py -i <inputfile> -n <line> -t <text>'
      sys.exit(2)
   for opt, arg in opts:
      if opt == '-h':
         print 'add-line.py -i <inputfile> -n <line> -t <text>'
         sys.exit()
      elif opt in ("-i"):
         inputfile = arg
      elif opt in ("-n"):
         line = arg
      elif opt in ("-t"):
         text = arg
   print 'Insert "', text,'" into line [', line, '] to file: ', inputfile
   insertNewLine(inputfile, int(line), text)

if __name__ == "__main__":
   main(sys.argv[1:])
