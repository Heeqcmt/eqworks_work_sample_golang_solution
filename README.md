# eqworks_work_sample_golang_solution

YuMing Zhang's attempted solution for EQ Works Product side, question 2b golang backend api


Counters are saved using SQLite 3. Keys are id(autoincrement), view and click. All values are type Integer.
Array of selected events are saved using a JSON file, because SQLite dones not support arrays. This could be improved by using a different storage.
