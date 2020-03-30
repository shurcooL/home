package fs

// Tree layout:
//
// 	root
// 	└── domain.com
// 	    └── path
// 	        └── _changes
// 	            ├── 1
// 	            │   ├── 0 - encoded change
// 	            │   ├── 1 - encoded timeline item
// 	            │   ├── 2 - encoded review
// 	            │   ├── 2a - encoded review comment
// 	            │   ├── 2b
// 	            │   ├── 2c
// 	            │   ├── 3
// 	            │   ├── 4
// 	            │   └── 5
// 	            └── 2
// 	                └── 0
