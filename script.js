
        function likePost(postId) {
            var likeCountElement = document.getElementById('post_' + postId + '_likes');
            var likeCount = parseInt(likeCountElement.textContent);
            likeCount++;
            likeCountElement.textContent = likeCount + ' Likes';

            // Send a request to update the like count on the server-side
            fetch('/like', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded'
                },
                body: 'post_id=' + postId
            });
        }

        function dislikePost(postId) {
            var dislikeCountElement = document.getElementById('post_' + postId + '_dislikes');
            var dislikeCount = parseInt(dislikeCountElement.textContent);
            dislikeCount++;
            dislikeCountElement.textContent = dislikeCount + ' Dislikes';

            // Send a request to update the dislike count on the server-side
            fetch('/dislike', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded'
                },
                body: 'post_id=' + postId
            });
        }

        function filterPosts() {
            var filterSelect = document.getElementById('filterSelect');
            var filterValue = filterSelect.value;

            var filterInput = document.getElementById('filterInput');
            var filterNumber = parseInt(filterInput.value);

            var posts = document.querySelectorAll('li');

            posts.forEach(function(post) {
                var likeCountElement = post.querySelector('.like-count');
                var dislikeCountElement = post.querySelector('.dislike-count');

                var likeCount = parseInt(likeCountElement.textContent);
                var dislikeCount = parseInt(dislikeCountElement.textContent);

                if (filterValue === 'all') {
                    post.style.display = 'block';
                } else if (filterValue === 'liked') {
                    if (likeCount > filterNumber) {
                        post.style.display = 'block';
                    } else {
                        post.style.display = 'none';
                    }
                } else if (filterValue === 'disliked') {
                    if (dislikeCount > filterNumber) {
                        post.style.display = 'block';
                    } else {
                        post.style.display = 'none';
                    }
                }
            });
        }
   