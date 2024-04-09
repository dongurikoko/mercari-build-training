class Solution {
public:
    int minEatingSpeed(vector<int>& piles, int h) {
        int l = 1,r = 1e9;
        while(l<r){
            int m = (l + r)/2;
            int sum = 0;
            for(int i=0;i<piles.size();i++){
                if(piles[i] % m == 0){
                    sum += piles[i] / m;
                }
                else{
                    sum += piles[i] / m + 1;
                }
            }
            if(sum <= h){
                r = m;
            }
            else{
                l = m + 1;
            }
        }
        return l;
    }
};
