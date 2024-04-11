class Solution { 
public:
    int lengthOfLongestSubstring(string s) {
        int answer = 1;
        if(s == ""){
            return 0;
        }
        for(int i=0;i<s.size();i++){
            vector<char> ans;
            ans.push_back(s[i]);
            map<char,int> m;
            m[s[i]] = 1;
            for(int j=i+1;j<s.size();j++){
                if(m[s[j]] == 0){
                    ans.push_back(s[j]);
                    m[s[j]] = 1;
                    if(j==s.size()-1){
                        j = j - 1;
                    }
                }
                else{
                    if(ans.size() > answer){
                        answer = ans.size();
                    }
                    break;
                }
            }
        }
        return answer;
    }
};
