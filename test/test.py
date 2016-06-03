import os
import os.path
import time
import random

curr_dir = os.getcwd()
print (curr_dir)
files_list = os.listdir()

for i in files_list: 
	if ".txt" in i:
		pass 
	else:
		files_list.remove(i)
		
print (files_list)

for i in range(random.randint(5,10)):
	print ("\n\n\niteration-->%s"%i)
	time.sleep(3)
	for f in files_list:
		print ("opening file %s"%f)
		time.sleep(5)
		#fl = open(f, "w")
		with open(f,"w") as fl : 

			#text = fl.read()
			text = f*random.randint(1,5)
			fl.write(text)
			#fl.close()


