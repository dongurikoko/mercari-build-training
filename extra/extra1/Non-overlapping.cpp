#include <bits/stdc++.h>
using namespace std;

class Solution {
public:
    static bool cmp(const pair<int,int>& a,const pair<int,int>& b){
        return a.second < b.second;
    }
    int eraseOverlapIntervals(vector<vector<int>>& intervals) {
        vector<pair<int,int>> v;
        for(int i=0;i<intervals.size();i++){
            v.push_back({intervals[i][0],intervals[i][1]});
        }

        // 第２引数が小さい順にソート
        sort(v.begin(),v.end(),cmp);

        int ans = 0;

        int end = v[0].second;

        for(int i=1;i<v.size();i++){
            if(v[i].first < end){
                ans++;
            }
            else{
                end = v[i].second;
            }
        }
    return ans;    
    }
};
